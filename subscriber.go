package rtsp2

import (
	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/rtsp"
	"m7s.live/engine/v4"
	"m7s.live/engine/v4/codec"
	"m7s.live/engine/v4/track"
)

type RTSPSubscriber struct {
	engine.Subscriber
	videoSender *core.Receiver
	audioSender *core.Receiver
	Conn        *rtsp.Conn `json:"-" yaml:"-"`
}

func (p *RTSPSubscriber) OnEvent(event any) {
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
		if p.videoSender == nil {
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
