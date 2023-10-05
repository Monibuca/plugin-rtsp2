package rtsp2

import (
	"go.uber.org/zap"
	"m7s.live/engine/v4"
	"m7s.live/engine/v4/config"
)

type RTSP2Config struct {
	config.Publish
	config.Pull
}

var conf RTSP2Config

var RTSP2Plugin = engine.InstallPlugin(&conf)

func (*RTSP2Config) OnEvent(event any) {
	switch v := event.(type) {
	case engine.FirstConfig:
		for streamPath, url := range conf.PullOnStart {
			if err := RTSP2Plugin.Pull(streamPath, url, new(RTSPPuller), 0); err != nil {
				RTSP2Plugin.Error("pull", zap.String("streamPath", streamPath), zap.String("url", url), zap.Error(err))
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
