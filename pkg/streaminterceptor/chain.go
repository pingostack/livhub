package streaminterceptor

type Chain struct {
	interceptors []Interceptor
}

func NewChain(interceptors []Interceptor) *Chain {
	return &Chain{
		interceptors: interceptors,
	}
}

func (c *Chain) Close() error {
	var errs []error
	for _, i := range c.interceptors {
		errs = append(errs, i.Close())
	}
	return flattenErrs(errs)
}

func (c *Chain) BindRemoteStream(md *Metadata, reader Reader) (*Metadata, Reader) {
	for _, i := range c.interceptors {
		md, reader = i.BindRemoteStream(md, reader)
	}
	return md, reader
}

func (c *Chain) BindLocalStream(md *Metadata, writer Writer) Writer {
	for _, i := range c.interceptors {
		writer = i.BindLocalStream(md, writer)
	}
	return writer
}

func (c *Chain) UnbindRemoteStream(md *Metadata) {
	for _, i := range c.interceptors {
		i.UnbindRemoteStream(md)
	}
}

func (c *Chain) UnbindLocalStream(md *Metadata) {
	for _, i := range c.interceptors {
		i.UnbindLocalStream(md)
	}
}
