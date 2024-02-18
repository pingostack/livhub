package streaminterceptor

type MediaKind string

const (
	MediaKindVideo       MediaKind = "video"
	MediaKindAudio       MediaKind = "audio"
	MediaKindApplication MediaKind = "application"
)

type Metadata struct {
	Kind MediaKind
}
