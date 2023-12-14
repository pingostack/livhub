package streaminterceptor

import (
	"io"
)

type Reader interface {
	Read([]byte, Attributes) (int, Attributes, error)
}

type Writer interface {
	Write(payload []byte, attributes Attributes) (int, error)
}

type Factory interface {
	NewInterceptor(id string) (Interceptor, error)
}

type Interceptor interface {
	BindRemoteStream(md *Metadata, reader Reader) (*Metadata, Reader)
	BindLocalStream(md *Metadata, writer Writer) Writer
	UnbindRemoteStream(md *Metadata)
	UnbindLocalStream(md *Metadata)
	io.Closer
}

type WriterFunc func(payload []byte, attributes Attributes) (int, error)
type ReaderFunc func(b []byte, attributes Attributes) (int, Attributes, error)

func (f WriterFunc) Write(payload []byte, attributes Attributes) (int, error) {
	return f(payload, attributes)
}

func (f ReaderFunc) Read(b []byte, attributes Attributes) (int, Attributes, error) {
	return f(b, attributes)
}
