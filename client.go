package rtsp2

import (
	"github.com/AlexxIT/go2rtc/pkg/core"
	"github.com/AlexxIT/go2rtc/pkg/rtsp"
	"go.uber.org/zap"
	"m7s.live/engine/v4"
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
