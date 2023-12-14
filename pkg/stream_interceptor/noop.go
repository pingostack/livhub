package streaminterceptor

type NoOp struct{}

func (i *NoOp) Close() error {
	return nil
}

func (i *NoOp) BindRemoteStream(md *Metadata, reader Reader) (*Metadata, Reader) {
	return md, reader
}

func (i *NoOp) BindLocalStream(md *Metadata, writer Writer) Writer {
	return writer
}

func (i *NoOp) UnbindRemoteStream(md *Metadata) {}

func (i *NoOp) UnbindLocalStream(md *Metadata) {}
