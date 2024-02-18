package deliver

import (
	"encoding/json"
)

type FrameCodec int

const (
	FrameCodecUnknown FrameCodec = 0 + iota
	FrameCodecI420
	FrameCodecVP8
	FrameCodecVP9
	FrameCodecH264
	FrameCodecH265
	FrameCodecAV1
	FrameCodecMSDK
	FrameCodecPCM_48000_2
	FrameCodecPCMU
	FrameCodecPCMA
	FrameCodecOpus
	FrameCodecISAC16
	FrameCodecISAC32
	FrameCodecILBC
	FrameCodecG722_16000_1
	FrameCodecG722_16000_2
	FrameCodecAAC
	FrameCodecAAC_48000_2
	FrameCodecAC3
	FrameCodecNellymoser
	FrameCodecData
)

func (fc FrameCodec) String() string {
	if fc == FrameCodecI420 {
		return "I420"
	} else if fc == FrameCodecVP8 {
		return "VP8"
	} else if fc == FrameCodecVP9 {
		return "VP9"
	} else if fc == FrameCodecH264 {
		return "H264"
	} else if fc == FrameCodecH265 {
		return "H265"
	} else if fc == FrameCodecAV1 {
		return "AV1"
	} else if fc == FrameCodecMSDK {
		return "MSDK"
	} else if fc == FrameCodecPCM_48000_2 {
		return "PCM_48000_2"
	} else if fc == FrameCodecPCMU {
		return "PCMU"
	} else if fc == FrameCodecPCMA {
		return "PCMA"
	} else if fc == FrameCodecOpus {
		return "Opus"
	} else if fc == FrameCodecISAC16 {
		return "ISAC16"
	} else if fc == FrameCodecISAC32 {
		return "ISAC32"
	} else if fc == FrameCodecILBC {
		return "ILBC"
	} else if fc == FrameCodecG722_16000_1 {
		return "G722_16000_1"
	} else if fc == FrameCodecG722_16000_2 {
		return "G722_16000_2"
	} else if fc == FrameCodecAAC {
		return "AAC"
	} else if fc == FrameCodecAAC_48000_2 {
		return "AAC_48000_2"
	} else if fc == FrameCodecAC3 {
		return "AC3"
	} else if fc == FrameCodecNellymoser {
		return "Nellymoser"
	} else if fc == FrameCodecData {
		return "Data"
	} else {
		return "Unknown"
	}
}

func (fc FrameCodec) IsAudio() bool {
	return fc >= FrameCodecPCM_48000_2 && fc <= FrameCodecNellymoser
}

func (fc FrameCodec) IsVideo() bool {
	return fc >= FrameCodecI420 && fc <= FrameCodecMSDK
}

func (fc FrameCodec) IsData() bool {
	return fc == FrameCodecData
}

func ConvFrameCodec(str string) FrameCodec {
	return FrameCodecUnknown
}

type PacketType int

const (
	PacketTypeUnknown PacketType = 0 + iota
	PacketTypeRtp
	PacketTypeRaw
	PacketTypeFlv
	PacketTypeRtmp
	PacketTypeTs
)

func (pt PacketType) String() string {
	if pt == PacketTypeRtp {
		return "Rtp"
	} else if pt == PacketTypeRaw {
		return "Raw"
	} else if pt == PacketTypeFlv {
		return "Flv"
	} else if pt == PacketTypeRtmp {
		return "Rtmp"
	} else if pt == PacketTypeTs {
		return "Ts"
	} else {
		return "Unknown"
	}
}

type VideoFrameSpecificInfo struct {
	Width      uint16 `json:"width"`
	Height     uint16 `json:"height"`
	IsKeyFrame bool   `json:"isKeyFrame"`
}

func (vfsi *VideoFrameSpecificInfo) String() string {
	jstr, _ := json.Marshal(vfsi)
	return string(jstr)
}

type AudioFrameSpecificInfo struct {
	IsRtpPacket bool   `json:"isRtpPacket"`
	NbSamples   uint32 `json:"nbSamples"`
	SampleRate  uint32 `json:"sampleRate"`
	Channels    uint8  `json:"channels"`
	Voice       uint8  `json:"voice"`
	AudioLevel  uint8  `json:"audioLevel"`
}

func (afsi *AudioFrameSpecificInfo) String() string {
	jstr, _ := json.Marshal(afsi)
	return string(jstr)
}

type FrameSpecificInfo interface {
	String() string
}

type Frame struct {
	Codec          FrameCodec
	PacketType     PacketType
	Payload        []byte
	Length         uint32
	TimeStamp      uint32
	AdditionalInfo FrameSpecificInfo
}

type AudioMetadata struct {
	Codec FrameCodec
	Type  PacketType
}

type VideoMetadata struct {
	Codec FrameCodec
	Type  PacketType
}

type DataMetadata struct {
	Codec FrameCodec
	Type  PacketType
}

type Metadata interface{}

type FeedbackType int
type FeedbackCmd int

const (
	FeedbackTypeUnknown FeedbackType = 0 + iota
	FeedbackTypeAudio
	FeedbackTypeVideo
	FeedbackTypeData
)

const (
	FeedbackCmdUnknown FeedbackCmd = 0 + iota
	FeedbackCmdKeyFrame
	FeedbackCmdNack
	FeedbackCmdPLI
	FeedbackCmdFIR
	FeedbackCmdSLI
	FeedbackCmdRPSI
)

type FeedbackMsg struct {
	Type FeedbackType
	Cmd  FeedbackCmd
}
