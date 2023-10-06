package rtsp2

import (
	"net"

	"github.com/AlexxIT/go2rtc/pkg/rtsp"
	"go.uber.org/zap"
	"m7s.live/engine/v4"
	"m7s.live/engine/v4/config"
)

type RTSP2Config struct {
	config.Publish
	config.Subscribe
	config.Pull
	config.Push
	config.TCP
}

var conf RTSP2Config

var RTSP2Plugin = engine.InstallPlugin(&conf)

func (c *RTSP2Config) OnEvent(event any) {
	switch v := event.(type) {
	case engine.FirstConfig:
		for streamPath, url := range conf.PullOnStart {
			if err := RTSP2Plugin.Pull(streamPath, url, new(RTSPPuller), 0); err != nil {
				RTSP2Plugin.Error("pull", zap.String("streamPath", streamPath), zap.String("url", url), zap.Error(err))
			}
		}
	case engine.SEpublish:
		if url, ok := conf.PushList[v.Target.Path]; ok {
			if err := RTSP2Plugin.Push(v.Target.Path, url, new(RTSPPusher), false); err != nil {
				RTSP2Plugin.Error("push", zap.String("streamPath", v.Target.Path), zap.String("url", url), zap.Error(err))
			}
		}
	case engine.InvitePublish: //按需拉流
		if url, ok := conf.PullOnSub[v.Target]; ok {
			if err := RTSP2Plugin.Pull(v.Target, url, new(RTSPPuller), 0); err != nil {
				RTSP2Plugin.Error("pull", zap.String("streamPath", v.Target), zap.String("url", url), zap.Error(err))
			}
		}
	}
}

func (c *RTSP2Config) ServeTCP(conn net.Conn) {
	server := rtsp.NewServer(conn)
	server.Listen(func(msg any) {
		switch msg {
		case rtsp.MethodPlay:
			var suber RTSPSubscriber
			if err := RTSP2Plugin.Subscribe(server.URL.Path, &suber); err != nil {
				server.Stop()
			}
		case rtsp.MethodRecord:
			var puber RTSPPublisher
			if err := RTSP2Plugin.Publish(server.URL.Path, &puber); err != nil {
				server.Stop()
			} else {
				puber.setTracks()
			}
		}
	})
}
