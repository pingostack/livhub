package streaminterceptor

type MediaKind string

const (
	MediaTypeVideo       MediaKind = "video"
	MediaTypeAudio       MediaKind = "audio"
	MediaTypeApplication MediaKind = "application"
)

type Metadata struct {
	Kind MediaKind
}
