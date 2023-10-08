package rtsp2

import (
	"encoding/hex"

	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/h264"
	"github.com/AlexxIT/go2rtc/pkg/h265"
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
		if m.Direction != core.DirectionRecvonly {
			continue
		}
		for _, c := range m.Codecs {
			var handler core.HandlerFunc
			switch c.Name {
			case core.CodecH264:
				if p.VideoTrack == nil {
					p.VideoTrack = track.NewH264(p.Stream, c.PayloadType)
					sps, pps := h264.GetParameterSet(c.FmtpLine)
					if len(sps) > 0 {
						p.VideoTrack.WriteSliceBytes(sps)
					}
					if len(pps) > 0 {
						p.VideoTrack.WriteSliceBytes(pps)
					}
				}
				handler = p.VideoTrack.WriteRTPPack
			case core.CodecH265:
				if p.VideoTrack == nil {
					p.VideoTrack = track.NewH265(p.Stream, c.PayloadType)
					vps, sps, pps := h265.GetParameterSet(c.FmtpLine)
					if len(vps) > 0 {
						p.VideoTrack.WriteSliceBytes(vps)
					}
					if len(sps) > 0 {
						p.VideoTrack.WriteSliceBytes(sps)
					}
					if len(pps) > 0 {
						p.VideoTrack.WriteSliceBytes(pps)
					}
				}
				handler = p.VideoTrack.WriteRTPPack
			case core.CodecAAC:
				if p.AudioTrack == nil {
					s := core.Between(c.FmtpLine, "config=", ";")
					asc, _ := hex.DecodeString(s)
					// var aacConfig mpeg4audio.AudioSpecificConfig
					// aacConfig.ChannelCount = int(c.Channels)
					// aacConfig.SampleRate = int(c.ClockRate)
					// aacConfig.Type = mpeg4audio.ObjectTypeAACLC
					// asc, _ := aacConfig.Marshal()
					aac := track.NewAAC(p.Stream, c.PayloadType, c.ClockRate)
					aac.WriteSequenceHead(append([]byte{0xAF, 0x00}, asc...))
					p.AudioTrack = aac
				}
				handler = p.AudioTrack.WriteRTPPack
			case core.CodecPCMA:
				if p.AudioTrack == nil {
					g711 := track.NewG711(p.Stream, true, c.PayloadType, c.ClockRate)
					g711.Channels = byte(c.Channels)
					p.AudioTrack = g711
				}
				handler = p.AudioTrack.WriteRTPPack
			case core.CodecPCMU:
				if p.AudioTrack == nil {
					g711 := track.NewG711(p.Stream, false, c.PayloadType, c.ClockRate)
					g711.Channels = byte(c.Channels)
					p.AudioTrack = g711
				}
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
