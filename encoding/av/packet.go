package av

const (
	AUDIO = 8
	VIDEO = 9
	META  = 18
)

type Packet struct {
	IsConfig   bool   // 是否是解码配置，如sps，pps
	IsKeyFrame bool   // 是否关键帧
	Type       uint8  // 包类型，8-audio，9-video，18-meta
	Timestamp  uint32 // 时间戳
	Payload    []byte // 负载
}

func (p *Packet) parseAudio() {
	if len(p.Payload) < 2 {
		return
	}
	format := p.Payload[0] >> 4
	if format == 10 { // AAC
		p.IsConfig = p.Payload[1] == 0
	}
}

func (p *Packet) parseVideo() {
	if len(p.Payload) < 2 {
		return
	}
	frameType := p.Payload[0] >> 4
	format := p.Payload[0] & 0x0F
	p.IsKeyFrame = frameType == 1
	if format == 7 { // AVC
		p.IsConfig = p.Payload[1] == 0
	}
}

func (p *Packet) IsAudio() bool {
	return p.Type == AUDIO
}

func (p *Packet) IsVideo() bool {
	return p.Type == VIDEO
}

func (p *Packet) IsMeta() bool {
	return p.Type == META
}

func MetaPack(timestamp uint32, data []byte) *Packet {
	return &Packet{
		Type:      META,
		Timestamp: timestamp,
		Payload:   data,
	}
}

func AudioPack(timestamp uint32, data []byte) *Packet {
	p := Packet{
		Type:      AUDIO,
		Timestamp: timestamp,
		Payload:   data,
	}
	p.parseAudio()
	return &p
}

func VideoPack(timestamp uint32, data []byte) *Packet {
	p := Packet{
		Type:      VIDEO,
		Timestamp: timestamp,
		Payload:   data,
	}
	p.parseVideo()
	return &p
}
