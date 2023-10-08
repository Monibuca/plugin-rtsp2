package rtsp2

import (
	"github.com/AlexxIT/go2rtc/pkg/rtsp"
	"m7s.live/engine/v4"
)

type RTSPPuller struct {
	RTSPPublisher
	engine.Puller
}

func (p *RTSPPuller) Disconnect() {
	if p.Conn != nil {
		p.Conn.Stop()
	}
}

func (p *RTSPPuller) Connect() (err error) {
	p.Conn = rtsp.NewClient(p.RemoteURL)
	return p.Conn.Dial()
}

func (p *RTSPPuller) Pull() (err error) {
	if err = p.Conn.Options(); err != nil {
		return
	}
	if err = p.Conn.Describe(); err != nil {
		return
	}
	p.setTracks()
	if p.AudioTrack == nil {
		p.Publisher.Config.PubAudio = false
	}
	if p.VideoTrack == nil {
		p.Publisher.Config.PubVideo = false
	}
	return p.Conn.Start()
}

type RTSPPusher struct {
	RTSPSubscriber
	engine.Pusher
}

func (p *RTSPPusher) Disconnect() {
	if p.Conn != nil {
		p.Conn.Stop()
	}
}

func (p *RTSPPusher) Connect() (err error) {
	p.Conn = rtsp.NewClient(p.RemoteURL)
	return p.Conn.Dial()
}

func (p *RTSPPusher) Push() (err error) {
	return p.Conn.Announce()
}
