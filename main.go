package rtsp2

import (
	"net"
	"net/http"
	"strconv"

	"github.com/AlexxIT/go2rtc/pkg/rtsp"
	"github.com/AlexxIT/go2rtc/pkg/tcp"
	"go.uber.org/zap"
	"m7s.live/engine/v4"
	"m7s.live/engine/v4/config"
	"m7s.live/engine/v4/util"
)

type RTSP2Config struct {
	config.HTTP
	config.Publish
	config.Subscribe
	config.Pull
	config.Push
	config.TCP
}

var conf = RTSP2Config{
	TCP: config.TCP{ListenAddr: ":554"},
}

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
		RTSP2Plugin.Debug("rtsp", zap.Any("msg", msg))
		switch msg := msg.(type) {
		case *tcp.Response:
			switch msg.Request.Method {
			case rtsp.MethodRecord, rtsp.MethodPlay:
				go server.Start()
			}
		case string:
			switch msg {
			case rtsp.MethodDescribe:
				var suber RTSPSubscriber
				suber.Conn = server
				if err := RTSP2Plugin.Subscribe(server.URL.Path, &suber); err != nil {
					server.Stop()
				}
			case rtsp.MethodAnnounce:
				var puber RTSPPublisher
				puber.Conn = server
				if err := RTSP2Plugin.Publish(server.URL.Path, &puber); err != nil {
					server.Stop()
				} else {
					puber.setTracks()
					if puber.AudioTrack == nil {
						puber.Publisher.Config.PubAudio = false
					}
					if puber.VideoTrack == nil {
						puber.Publisher.Config.PubVideo = false
					}
				}
			}
		}
	})
	server.Accept()
}

func filterStreams() (ss []*engine.Stream) {
	engine.Streams.Range(func(key string, s *engine.Stream) {
		switch s.Publisher.(type) {
		case *RTSPPublisher, *RTSPPuller:
			ss = append(ss, s)
		}
	})
	return
}

func (*RTSP2Config) API_list(w http.ResponseWriter, r *http.Request) {
	util.ReturnFetchValue(filterStreams, w, r)
}

func (*RTSP2Config) API_Pull(rw http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	save, _ := strconv.Atoi(query.Get("save"))
	err := RTSP2Plugin.Pull(query.Get("streamPath"), query.Get("target"), new(RTSPPuller), save)
	if err != nil {
		util.ReturnError(util.APIErrorQueryParse, err.Error(), rw, r)
	} else {
		util.ReturnOK(rw, r)
	}
}

func (*RTSP2Config) API_Push(rw http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	err := RTSP2Plugin.Push(query.Get("streamPath"), query.Get("target"), new(RTSPPusher), query.Has("save"))
	if err != nil {
		util.ReturnError(util.APIErrorQueryParse, err.Error(), rw, r)
	} else {
		util.ReturnOK(rw, r)
	}
}
