package forwarder

import "sync"

type FrameDestination struct {
	src             IFrameSource
	mutex           sync.Mutex
	onSourceChanged func()
}

func (fd *FrameDestination) WriteFrame(frame *Frame) error {
	return nil
}

func (fd *FrameDestination) SetFrameSource(source IFrameSource) {
	fd.mutex.Lock()
	fd.src = source
	fd.mutex.Unlock()

	if fd.onSourceChanged != nil {
		fd.onSourceChanged()
	}
}

func (fd *FrameDestination) UnsetFrameSource() {
	fd.mutex.Lock()
	defer fd.mutex.Unlock()

	fd.src = nil
}

func (fd *FrameDestination) DeliverFeedback(fb *FeedbackMsg) {
	fd.mutex.Lock()
	defer fd.mutex.Unlock()

	if fd.src != nil {
		fd.src.WriteFeedback(fb)
	}
}

func (fd *FrameDestination) OnSrouceChanged(fn func()) {
	fd.onSourceChanged = fn
}
