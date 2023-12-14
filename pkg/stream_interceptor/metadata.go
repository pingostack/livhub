package streaminterceptor

type MediaType string

const (
	MediaVideo       MediaType = "video"
	MediaAudio       MediaType = "audio"
	MediaApplication MediaType = "application"
)

type Metadata struct {
	MediaType MediaType
}
