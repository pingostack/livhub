package forwarder

import "sync"

type FrameSource struct {
	pt      PacketType
	format  FrameFormat
	mutex   sync.Mutex
	dest    []IFrameDestination
	destMap map[string]IFrameDestination
}

func NewFrameSource(pt PacketType, format FrameFormat) *FrameSource {
	return &FrameSource{
		pt:     pt,
		format: format,
	}
}

func (fs *FrameSource) WriteFeedback(fb *FeedbackMsg) {
	return
}

func (fs *FrameSource) AddDestination(dest IFrameDestination) {
	dest.SetFrameSource(fs)

	fs.mutex.Lock()
	fs.dest = append(fs.dest, dest)
	fs.mutex.Unlock()
}

func (fs *FrameSource) RemoveDestination(dest IFrameDestination) {
	dest.UnsetFrameSource()

	fs.mutex.Lock()
	for i, d := range fs.dest {
		if d == dest {
			fs.dest = append(fs.dest[:i], fs.dest[i+1:]...)
			return
		}
	}
	fs.mutex.Unlock()
}

func (fs *FrameSource) DeliverFrame(frame *Frame) {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	for _, dest := range fs.dest {
		dest.WriteFrame(frame)
	}
}

func (fs *FrameSource) CleanDestination() {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	for _, dest := range fs.dest {
		dest.UnsetFrameSource()
	}

	fs.dest = nil
}

func (fs *FrameSource) FrameFormat() FrameFormat {
	return fs.format
}

func (fs *FrameSource) PacketType() PacketType {
	return fs.pt
}

func (fs *FrameSource) GetFrameDestination(id string) IFrameDestination {
	fs.mutex.Lock()
	defer fs.mutex.Unlock()

	return fs.destMap[id]
}
