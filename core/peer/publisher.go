package peer

import "github.com/pingostack/livhub/pkg/avframe"

type Publisher interface {
	avframe.ReadCloser
	// ID returns the unique identifier of the publisher
	ID() string
	Format() avframe.FmtType
	Metadata() avframe.Metadata
}
