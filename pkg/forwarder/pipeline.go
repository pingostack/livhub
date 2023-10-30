package forwarder

type FrameKind uint8

const (
	FrameKindVideo FrameKind = 1
	FrameKindAudio           = 2
	FrameKindData            = 4
)

type PacketType uint8

const (
	PacketTypeRtp PacketType = iota + 1
	PacketTypeRtcp
	PacketTypeRtmp
	PacketTypeFLV
)

func (pt PacketType) String() string {
	switch pt {
	case PacketTypeRtp:
		return "rtp"
	case PacketTypeRtcp:
		return "rtcp"
	case PacketTypeRtmp:
		return "rtmp"
	case PacketTypeFLV:
		return "flv"
	default:
		return "unknown"
	}
}

type FrameFormat uint8

const (
	FrameFormatUnknown FrameFormat = iota
	FrameFormatVP8
	FrameFormatVP9
	FrameFormatH264
	FrameFormatH265
	FrameFormatVP8SVC
	FrameFormatVP9SVC
	FrameFormatH264SVC
	FrameFormatH265SVC
	FrameFormatAV1
	FrameFormatOpus
	FrameFormatG722
	FrameFormatPCMU
	FrameFormatPCMA
	FrameFormatAAC
)

func (ct FrameFormat) String() string {
	switch ct {
	case FrameFormatVP8:
		return "vp8"
	case FrameFormatVP9:
		return "vp9"
	case FrameFormatH264:
		return "h264"
	case FrameFormatH265:
		return "h265"
	case FrameFormatVP8SVC:
		return "vp8svc"
	case FrameFormatVP9SVC:
		return "vp9svc"
	case FrameFormatH264SVC:
		return "h264svc"
	case FrameFormatH265SVC:
		return "h265svc"
	case FrameFormatAV1:
		return "av1"
	case FrameFormatOpus:
		return "opus"
	case FrameFormatG722:
		return "g722"
	case FrameFormatPCMU:
		return "pcmu"
	case FrameFormatPCMA:
		return "pcma"
	case FrameFormatAAC:
		return "aac"
	default:
		return "unknown"
	}
}

type FeedbackType uint16

const (
	FeedbackTypeNack FeedbackType = iota + 1
	FeedbackTypePLI
)

type FeedbackMsg struct {
	Type FeedbackType
	Data interface{}
}

type VideoRotation int

const (
	VIDEO_ROTATION_0     VideoRotation = 0
	VIDEO_ROTATION_90                  = 90
	VIDEO_ROTATION_180                 = 180
	VIDEO_ROTATION_270                 = 270
	VIDEO_ROTATION_UNSET               = 360
)

type VideoFrameSpecificInfo struct {
	Width      uint16
	Height     uint16
	IsKeyFrame bool
	Rotation   VideoRotation
	Layer      uint8
}

type AudioFrameSpecificInfo struct {
	NbSamples  uint32
	SampleRate uint32
	Channels   uint8
}

type Frame struct {
	PT        PacketType
	Format    FrameFormat
	Layer     int8
	Timestamp uint64
	VFSI      *VideoFrameSpecificInfo
	AFSI      *AudioFrameSpecificInfo
	Payload   interface{}
}

type IFrameSource interface {
	WriteFeedback(fb *FeedbackMsg)
	AddDestination(dest IFrameDestination)
	RemoveDestination(dest IFrameDestination)
	CleanDestination()
	DeliverFrame(frame *Frame)
	FrameFormat() FrameFormat
	PacketType() PacketType
	GetFrameDestination(id string) IFrameDestination
}

type IFrameDestination interface {
	WriteFrame(frame *Frame) error
	SetFrameSource(source IFrameSource)
	OnSrouceChanged(fn func())
	UnsetFrameSource()
	DeliverFeedback(fb *FeedbackMsg)
	FrameFormat() FrameFormat
	PacketType() PacketType
}
