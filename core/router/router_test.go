package router

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/pingostack/livhub/core/peer"
	"github.com/pingostack/livhub/core/plugin"
	"github.com/pingostack/livhub/core/stream"
	"github.com/pingostack/livhub/pkg/avframe"
	"github.com/pingostack/livhub/pkg/errcode"
	"github.com/stretchr/testify/assert"
)

type testPublisher struct {
	id        string
	format    avframe.FmtType
	processor avframe.Processor
	closed    bool
	mu        sync.Mutex
}

func (p *testPublisher) ID() string {
	return p.id
}

func (p *testPublisher) Format() avframe.FmtType {
	return p.format
}

func (p *testPublisher) OnActive(processor avframe.Processor) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.processor = processor
}

func (p *testPublisher) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	return nil
}

func (p *testPublisher) Read() (*avframe.Frame, error) {
	return nil, nil
}

func (p *testPublisher) Metadata() avframe.Metadata {
	return avframe.Metadata{
		AudioCodecType: avframe.CodecTypeAAC,
		VideoCodecType: avframe.CodecTypeH264,
		FmtType:        p.format,
	}
}

type testSubscriber struct {
	id        string
	format    avframe.FmtType
	processor avframe.Processor
	closed    bool
	mu        sync.Mutex
}

func (s *testSubscriber) ID() string {
	return s.id
}

func (s *testSubscriber) Format() avframe.FmtType {
	return s.format
}

func (s *testSubscriber) OnActive(processor avframe.Processor) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.processor = processor
}

func (s *testSubscriber) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.closed = true
	return nil
}

func (s *testSubscriber) Write(frame *avframe.Frame) error {
	return nil
}

func (s *testSubscriber) AudioCodecSupported() []avframe.CodecType {
	return []avframe.CodecType{avframe.CodecTypeAAC}
}

func (s *testSubscriber) VideoCodecSupported() []avframe.CodecType {
	return []avframe.CodecType{avframe.CodecTypeH264}
}

type mockDemuxer struct {
	avframe.NoopProcessor
	fmtType avframe.FmtType
}

func (m *mockDemuxer) GetFormatType() avframe.FmtType {
	return m.fmtType
}

type mockMuxer struct {
	avframe.NoopProcessor
	fmtType avframe.FmtType
}

func (m *mockMuxer) GetFormatType() avframe.FmtType {
	return m.fmtType
}

func init() {
	// Register RTMP demuxer
	plugin.RegisterDemuxerPlugin(avframe.FormatRtmp, func(ctx context.Context, metadata avframe.Metadata) (plugin.Demuxer, error) {
		return &mockDemuxer{fmtType: avframe.FormatRtmp}, nil
	})

	// Register RTP/RTCP demuxer
	plugin.RegisterDemuxerPlugin(avframe.FormatRtpRtcp, func(ctx context.Context, metadata avframe.Metadata) (plugin.Demuxer, error) {
		return &mockDemuxer{fmtType: avframe.FormatRtpRtcp}, nil
	})

	// Register RTMP muxer
	plugin.RegisterMuxerPlugin(avframe.FormatRtmp, func(ctx context.Context, metadata avframe.Metadata) (plugin.Muxer, error) {
		return &mockMuxer{fmtType: avframe.FormatRtmp}, nil
	})

	// Register RTP/RTCP muxer
	plugin.RegisterMuxerPlugin(avframe.FormatRtpRtcp, func(ctx context.Context, metadata avframe.Metadata) (plugin.Muxer, error) {
		return &mockMuxer{fmtType: avframe.FormatRtpRtcp}, nil
	})
}

func TestRouter(t *testing.T) {
	// 测试中间件调用计数
	var publishCount, subscribeCount, streamCount, pullCount, pushCount int
	var mu sync.Mutex

	// 注册中间件
	RegisterPublishMiddleware(func(ctx context.Context, publisher peer.Publisher, stage Stage) error {
		mu.Lock()
		publishCount++
		mu.Unlock()
		return nil
	})

	RegisterSubscribeMiddleware(func(ctx context.Context, subscriber peer.Subscriber, stage Stage) error {
		mu.Lock()
		subscribeCount++
		mu.Unlock()
		return nil
	})

	RegisterStreamMiddleware(func(ctx context.Context, s *stream.Stream, stage Stage) error {
		mu.Lock()
		streamCount++
		mu.Unlock()
		return nil
	})

	RegisterPullMiddleware(func(ctx context.Context, s *stream.Stream, stage Stage) error {
		mu.Lock()
		pullCount++
		mu.Unlock()
		return nil
	})

	RegisterPushMiddleware(func(ctx context.Context, s *stream.Stream, stage Stage) error {
		mu.Lock()
		pushCount++
		mu.Unlock()
		return nil
	})

	t.Run("basic workflow", func(t *testing.T) {
		ctx := context.Background()
		router := NewRouter(ctx, "test")
		defer router.Close()

		// 测试发布
		pub := &testPublisher{
			id:     "test_pub",
			format: avframe.FormatRtmp,
		}
		err := router.Publish(pub)
		assert.NoError(t, err)

		// 测试订阅
		sub := &testSubscriber{
			id:     "test_sub",
			format: avframe.FormatRtpRtcp,
		}
		err = router.Subscribe(sub)
		assert.NoError(t, err)

		// 等待处理器设置
		time.Sleep(100 * time.Millisecond)

		// 验证订阅者是否收到处理器
		sub.mu.Lock()
		assert.NotNil(t, sub.processor)
		sub.mu.Unlock()

		// 测试取消发布
		err = router.Unpublish(pub)
		assert.NoError(t, err)

		// 测试取消订阅
		err = router.Unsubscribe(sub)
		assert.NoError(t, err)

		// 验证中间件调用次数
		mu.Lock()
		assert.Greater(t, publishCount, 0, "publish middleware should be called")
		assert.Greater(t, subscribeCount, 0, "subscribe middleware should be called")
		assert.Greater(t, streamCount, 0, "stream middleware should be called")
		assert.Greater(t, pullCount, 0, "pull middleware should be called")
		mu.Unlock()
	})

	t.Run("no publisher", func(t *testing.T) {
		ctx := context.Background()
		router := NewRouter(ctx, "test_no_pub")
		defer router.Close()

		// 测试订阅（无发布者）
		sub := &testSubscriber{
			id:     "test_sub",
			format: avframe.FormatRtpRtcp,
		}
		err := router.Subscribe(sub)
		assert.NoError(t, err)

		// 验证订阅者未收到处理器
		time.Sleep(100 * time.Millisecond)
		sub.mu.Lock()
		assert.Nil(t, sub.processor)
		sub.mu.Unlock()
	})

	t.Run("error cases", func(t *testing.T) {
		ctx := context.Background()
		router := NewRouter(ctx, "test_error")
		
		// 测试重复发布
		pub := &testPublisher{
			id:     "test_pub",
			format: avframe.FormatRtmp,
		}
		err := router.Publish(pub)
		assert.NoError(t, err)

		err = router.Publish(pub)
		assert.Error(t, err)
		assert.True(t, errcode.Is(err, errcode.ErrPublisherAlreadySet))

		// 关闭路由后的操作
		router.Close()
		err = router.Publish(pub)
		assert.Error(t, err)
		
		err = router.Subscribe(&testSubscriber{
			id:     "test_sub",
			format: avframe.FormatRtpRtcp,
		})
		assert.Error(t, err)
	})
}
