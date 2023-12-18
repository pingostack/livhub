package streaminterceptor

type MediaType string

const (
	MediaTypeVideo       MediaType = "video"
	MediaTypeAudio       MediaType = "audio"
	MediaTypeApplication MediaType = "application"
)

type Metadata struct {
	MediaType MediaType
}
