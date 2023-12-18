package forwarder

import (
	"context"

	"github.com/pingostack/neon/pkg/parallel"
	streaminterceptor "github.com/pingostack/neon/pkg/stream_interceptor"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type StreamFormat struct {
	ctx            context.Context
	name           string
	logger         *logrus.Entry
	interceptor    streaminterceptor.Interceptor
	originMetadata *streaminterceptor.Metadata
	originReader   streaminterceptor.Reader
	metadata       *streaminterceptor.Metadata
	writers        []streaminterceptor.Writer
}

func NewStreamFormat(ctx context.Context,
	name string,
	metadata *streaminterceptor.Metadata,
	r *streaminterceptor.Registry,
	reader streaminterceptor.Reader,
	logger *logrus.Entry) (*StreamFormat, error) {
	f := &StreamFormat{
		ctx:            ctx,
		name:           name,
		originMetadata: metadata,
		logger:         logger.WithField("format", name),
	}

	if err := f.setInterceptor(r, reader); err != nil {
		f.logger.Errorf("NewStreamFormat(%v) failed: %v", name, err)
		return nil, errors.Wrapf(err, "failed to set interceptor for format %v", name)
	}

	go f.run()

	return f, nil
}

func (f *StreamFormat) Name() string {
	return f.name
}

func (f *StreamFormat) setInterceptor(r *streaminterceptor.Registry, reader streaminterceptor.Reader) error {
	f.logger.Infof("StreamFormat.SetInterceptor(%v)", r)
	i, err := r.Build(f.name)
	if err != nil {
		f.logger.Errorf("StreamFormat.SetInterceptor(%v) failed: %v", r, err)
		return errors.Wrapf(err, "failed to build interceptor for format %v", f.name)
	}

	f.logger.Infof("StreamFormat.SetInterceptor(%v) success", r)

	f.interceptor = i
	f.originReader = reader

	return nil
}

func (f *StreamFormat) run() {
	f.logger.Infof("StreamFormat.run()")
	defer f.logger.Infof("StreamFormat.run() done")
	metadata, reader := f.interceptor.BindRemoteStream(f.originMetadata, f.originReader)
	f.metadata = metadata

	go func() {
		<-f.ctx.Done()
		f.logger.Infof("StreamFormat.run() context done")
	}()

	for {
		buf := make([]byte, 1024)
		_, _, err := reader.Read(buf, nil)
		if err != nil {
			f.logger.Errorf("StreamFormat.run() failed: %v", err)
			return
		}

		parallel.ParallelExec(f.writers, 100, 2, func(w streaminterceptor.Writer) {
			w.Write(buf, nil)
		})
	}
}

func (f *StreamFormat) Metadata() *streaminterceptor.Metadata {
	return f.metadata
}
