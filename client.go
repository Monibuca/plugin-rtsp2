package rtsp2

import (
	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/rtsp"
	"go.uber.org/zap"
	"m7s.live/engine/v4"
	"m7s.live/engine/v4/codec"
	"m7s.live/engine/v4/track"
)

type RTSPClient struct {
	*rtsp.Conn `json:"-" yaml:"-"`
	// DialContext func(ctx context.Context, network, address string) (net.Conn, error) `json:"-" yaml:"-"`
}

type RTSPPuller struct {
	engine.Publisher
	engine.Puller
	RTSPClient
}

func (p *RTSPClient) Disconnect() {
	if p.Conn != nil {
		p.Conn.Close()
	}
}

func (p *RTSPPuller) Connect() (err error) {
	p.Conn = rtsp.NewClient(p.RemoteURL)
	p.SetIO(p.Conn)
	return p.Conn.Dial()
}

func (p *RTSPPuller) Pull() (err error) {
	if err = p.Options(); err != nil {
		return
	}
	if err = p.Describe(); err != nil {
		return
	}
	p.setTracks()
	return p.Start()
}

func (p *RTSPPuller) setTracks() {
	for _, m := range p.Conn.Medias {
		for _, c := range m.Codecs {
			sender := core.NewSender(m, c)
			rec, err := p.Conn.GetTrack(m, c)
			if err != nil {
				p.Error("get track", zap.Error(err))
				continue
			}
			switch c.Name {
			case core.CodecH264:
				p.VideoTrack = track.NewH264(p.Stream, c.PayloadType)
				sender.Handler = p.VideoTrack.WriteRTPPack
			case core.CodecH265:
				p.VideoTrack = track.NewH265(p.Stream, c.PayloadType)
				sender.Handler = p.VideoTrack.WriteRTPPack
			case core.CodecAAC:
				p.AudioTrack = track.NewAAC(p.Stream, c.PayloadType)
				sender.Handler = p.AudioTrack.WriteRTPPack
			case core.CodecPCMA:
				p.AudioTrack = track.NewG711(p.Stream, true, c.PayloadType)
				sender.Handler = p.AudioTrack.WriteRTPPack
			case core.CodecPCMU:
				p.AudioTrack = track.NewG711(p.Stream, false, c.PayloadType)
				sender.Handler = p.AudioTrack.WriteRTPPack
			}
			sender.HandleRTP(rec)
		}
	}
}

type RTSPPusher struct {
	engine.Subscriber
	engine.Pusher
	RTSPClient
	videoSender *core.Receiver
	audioSender *core.Receiver
}

func (p *RTSPPusher) OnEvent(event any) {
	switch v := event.(type) {
	case *track.Audio:
		if p.audioSender != nil {
			break
		}
		var c *core.Codec
		var media *core.Media
		switch v.CodecID {
		case codec.CodecID_AAC:
			c = &core.Codec{
				Name:        core.CodecAAC,
				ClockRate:   v.SampleRate,
				Channels:    uint16(v.Channels),
				PayloadType: v.PayloadType,
			}
		case codec.CodecID_PCMA:
			c = &core.Codec{
				Name:        core.CodecPCMA,
				ClockRate:   v.SampleRate,
				Channels:    uint16(v.Channels),
				PayloadType: v.PayloadType,
			}
		case codec.CodecID_PCMU:
			c = &core.Codec{
				Name:        core.CodecPCMU,
				ClockRate:   v.SampleRate,
				Channels:    uint16(v.Channels),
				PayloadType: v.PayloadType,
			}
		}
		media = &core.Media{
			Kind:      "audio",
			Direction: "sendonly",
			Codecs:    []*core.Codec{c},
		}
		if p.videoSender  == nil {
			media.ID = "0"
		} else {
			media.ID = "1"
		}
		p.audioSender = core.NewReceiver(media, c)
		p.Conn.AddTrack(media, c, p.audioSender)
		p.AddTrack(v)
	case *track.Video:
		if p.videoSender != nil {
			break
		}
		var c *core.Codec
		var media *core.Media
		switch v.CodecID {
		case codec.CodecID_H264:
			c = &core.Codec{
				Name:        core.CodecH264,
				ClockRate:   v.SampleRate,
				PayloadType: v.PayloadType,
			}
		case codec.CodecID_H265:
			c = &core.Codec{
				Name:        core.CodecH265,
				ClockRate:   v.SampleRate,
				PayloadType: v.PayloadType,
			}
		}
		media = &core.Media{
			Kind:      "video",
			Direction: "sendonly",
			Codecs:    []*core.Codec{c},
		}
		if p.audioSender == nil {
			media.ID = "0"
		} else {
			media.ID = "1"
		}
		p.videoSender = core.NewReceiver(media, c)
		p.Conn.AddTrack(media, c, p.videoSender)
		p.AddTrack(v)
	case engine.VideoRTP:
		p.videoSender.WriteRTP(v.Packet)
	case engine.AudioRTP:
		p.audioSender.WriteRTP(v.Packet)
	default:
		p.Subscriber.OnEvent(event)
	}
}

func (p *RTSPPusher) Connect() (err error) {
	p.Conn = rtsp.NewClient(p.RemoteURL)
	p.SetIO(p.Conn)
	return p.Conn.Dial()
}

func (p *RTSPPuller) Push() (err error) {
	return p.Announce()
}
