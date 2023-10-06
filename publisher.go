package rtsp2

import (
	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/rtsp"
	"go.uber.org/zap"
	"m7s.live/engine/v4"
	"m7s.live/engine/v4/track"
)

type RTSPPublisher struct {
	engine.Publisher
	Conn *rtsp.Conn `json:"-" yaml:"-"`
}

func (p *RTSPPublisher) OnEvent(event any) {
	switch event.(type) {
	case engine.SEclose, engine.SEKick:
		p.Conn.Stop()
	}
	p.Publisher.OnEvent(event)
}

func (p *RTSPPublisher) setTracks() {
	for _, m := range p.Conn.Medias {
		for _, c := range m.Codecs {
			var handler core.HandlerFunc
			switch c.Name {
			case core.CodecH264:
				p.VideoTrack = track.NewH264(p.Stream, c.PayloadType)
				handler = p.VideoTrack.WriteRTPPack
			case core.CodecH265:
				p.VideoTrack = track.NewH265(p.Stream, c.PayloadType)
				handler = p.VideoTrack.WriteRTPPack
			case core.CodecAAC:
				p.AudioTrack = track.NewAAC(p.Stream, c.PayloadType)
				handler = p.AudioTrack.WriteRTPPack
			case core.CodecPCMA:
				p.AudioTrack = track.NewG711(p.Stream, true, c.PayloadType)
				handler = p.AudioTrack.WriteRTPPack
			case core.CodecPCMU:
				p.AudioTrack = track.NewG711(p.Stream, false, c.PayloadType)
				handler = p.AudioTrack.WriteRTPPack
			}
			if handler != nil {
				sender := core.NewSender(m, c)
				rec, err := p.Conn.GetTrack(m, c)
				if err != nil {
					p.Error("get track", zap.Error(err))
					continue
				}
				sender.Handler = handler
				sender.HandleRTP(rec)
			}
		}
	}
}
