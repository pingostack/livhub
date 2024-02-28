package deliver

import (
	"encoding/json"
	"strings"
)

type CodecType int

const (
	CodecTypeNone CodecType = 0 + iota
	CodecTypeI420
	CodecTypeVP8
	CodecTypeVP9
	CodecTypeH264
	CodecTypeH265
	CodecTypeAV1
	CodecTypeMSDK
	CodecTypePCM_48000_2
	CodecTypePCMU
	CodecTypePCMA
	CodecTypeOpus
	CodecTypeISAC16
	CodecTypeISAC32
	CodecTypeILBC
	CodecTypeG722_16000_1
	CodecTypeG722_16000_2
	CodecTypeAAC
	CodecTypeAAC_48000_2
	CodecTypeAC3
	CodecTypeNellymoser
)

func (fc CodecType) String() string {
	if fc == CodecTypeI420 {
		return "I420"
	} else if fc == CodecTypeVP8 {
		return "VP8"
	} else if fc == CodecTypeVP9 {
		return "VP9"
	} else if fc == CodecTypeH264 {
		return "H264"
	} else if fc == CodecTypeH265 {
		return "H265"
	} else if fc == CodecTypeAV1 {
		return "AV1"
	} else if fc == CodecTypeMSDK {
		return "MSDK"
	} else if fc == CodecTypePCM_48000_2 {
		return "PCM_48000_2"
	} else if fc == CodecTypePCMU {
		return "PCMU"
	} else if fc == CodecTypePCMA {
		return "PCMA"
	} else if fc == CodecTypeOpus {
		return "Opus"
	} else if fc == CodecTypeISAC16 {
		return "ISAC16"
	} else if fc == CodecTypeISAC32 {
		return "ISAC32"
	} else if fc == CodecTypeILBC {
		return "ILBC"
	} else if fc == CodecTypeG722_16000_1 {
		return "G722_16000_1"
	} else if fc == CodecTypeG722_16000_2 {
		return "G722_16000_2"
	} else if fc == CodecTypeAAC {
		return "AAC"
	} else if fc == CodecTypeAAC_48000_2 {
		return "AAC_48000_2"
	} else if fc == CodecTypeAC3 {
		return "AC3"
	} else if fc == CodecTypeNellymoser {
		return "Nellymoser"
	} else {
		return "None"
	}
}

func (fc CodecType) IsAudio() bool {
	return fc >= CodecTypePCM_48000_2 && fc <= CodecTypeNellymoser
}

func (fc CodecType) IsVideo() bool {
	return fc >= CodecTypeI420 && fc <= CodecTypeMSDK
}

func (fc CodecType) IsData() bool {
	return fc == CodecTypeNone
}

func ConvCodecType(str string) CodecType {
	if strings.EqualFold(str, "I420") {
		return CodecTypeI420
	} else if strings.EqualFold(str, "VP8") {
		return CodecTypeVP8
	} else if strings.EqualFold(str, "VP9") {
		return CodecTypeVP9
	} else if strings.EqualFold(str, "H264") {
		return CodecTypeH264
	} else if strings.EqualFold(str, "H265") {
		return CodecTypeH265
	} else if strings.EqualFold(str, "AV1") {
		return CodecTypeAV1
	} else if strings.EqualFold(str, "MSDK") {
		return CodecTypeMSDK
	} else if strings.EqualFold(str, "PCM_48000_2") {
		return CodecTypePCM_48000_2
	} else if strings.EqualFold(str, "PCMU") {
		return CodecTypePCMU
	} else if strings.EqualFold(str, "PCMA") {
		return CodecTypePCMA
	} else if strings.EqualFold(str, "Opus") {
		return CodecTypeOpus
	} else if strings.EqualFold(str, "ISAC16") {
		return CodecTypeISAC16
	} else if strings.EqualFold(str, "ISAC32") {
		return CodecTypeISAC32
	} else if strings.EqualFold(str, "ILBC") {
		return CodecTypeILBC
	} else if strings.EqualFold(str, "G722_16000_1") {
		return CodecTypeG722_16000_1
	} else if strings.EqualFold(str, "G722_16000_2") {
		return CodecTypeG722_16000_2
	} else if strings.EqualFold(str, "AAC") {
		return CodecTypeAAC
	} else if strings.EqualFold(str, "AAC_48000_2") {
		return CodecTypeAAC_48000_2
	} else if strings.EqualFold(str, "AC3") {
		return CodecTypeAC3
	} else if strings.EqualFold(str, "Nellymoser") {
		return CodecTypeNellymoser
	} else {
		return CodecTypeNone
	}
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
	Codec          CodecType
	PacketType     PacketType
	Payload        []byte
	Length         uint32
	TimeStamp      uint32
	AdditionalInfo FrameSpecificInfo
}

type AudioMetadata struct {
	Codec      CodecType
	Type       PacketType
	SampleRate int
	Channels   int
}

type VideoMetadata struct {
	Codec  CodecType
	Type   PacketType
	Width  int
	Height int
	FPS    int
}

type DataMetadata struct {
	Codec CodecType
	Type  PacketType
}

type Metadata struct {
	Audio *AudioMetadata
	Video *VideoMetadata
	Data  *DataMetadata
}

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
