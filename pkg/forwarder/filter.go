package forwarder

import (
	"io"
	"sync"
	"time"

	"github.com/gammazero/deque"
	"github.com/let-light/neon/pkg/utils"
)

type IFilter interface {
	IFrameSource
	IFrameDestination
}

type FilterNode struct {
	Filter IFilter
	Next   *FilterNode
}

type FilterChain struct {
	Head       *FilterNode
	Tail       *FilterNode
	extPackets deque.Deque
	closed     utils.AtomicBool
	mutex      sync.Mutex
	closeOnce  sync.Once
}

func NewFilterChain() *FilterChain {
	fc := &FilterChain{
		Head: nil,
		Tail: nil,
	}

	fc.extPackets.SetMinCapacity(7)

	return fc
}

func (fc *FilterChain) Append(filter IFilter) {
	node := &FilterNode{
		Filter: filter,
	}

	if fc.Head == nil {
		fc.Head = node
		fc.Tail = node
	} else {
		fc.Tail.Next = node
		fc.Tail = node
	}
}

func (fc *FilterChain) WriteFrame(frame *Frame) {
	node := fc.Head
	fp := frame
	var err error

	for node != nil {
		if e := node.Filter.WriteFrame(fp); e != nil {
			return
		}

		fp, err = node.Filter.ReadFrame()
		if err != nil {
			return
		}

		node = node.Next
	}

	fc.mutex.Lock()
	fc.extPackets.PushBack(fp)
	fc.mutex.Unlock()
}

func (fc *FilterChain) ReadFrame() (*Frame, error) {
	for {
		if fc.closed.Get() {
			return nil, io.EOF
		}
		fc.mutex.Lock()
		if fc.extPackets.Len() > 0 {
			fp := fc.extPackets.PopFront().(*Frame)
			fc.mutex.Unlock()
			return fp, nil
		}
		fc.mutex.Unlock()
		time.Sleep(10 * time.Millisecond)
	}
}

func (fc *FilterChain) Close() {
	fc.closeOnce.Do(func() {
		fc.mutex.Lock()
		defer fc.mutex.Unlock()

		if fc.closed.Get() {
			return
		}

		node := fc.Head
		for node != nil {
			node.Filter.Close()
			node = node.Next
		}

		fc.closed.Set(true)
	})
}

func (fc *FilterChain) RequestKeyFrame() error {
	node := fc.Head

	for node != nil {
		if e := node.Filter.RequestKeyFrame(); e != nil {
			return e
		}

		node = node.Next
	}

	return nil
}

func (fc *FilterChain) FrameKind() FrameKind {
	return fc.Tail.Filter.FrameKind()
}

func (fc *FilterChain) PacketType() PacketType {
	return fc.Tail.Filter.PacketType()
}

func (fc *FilterChain) FrameFormat() FrameFormat {
	return fc.Tail.Filter.FrameFormat()
}
