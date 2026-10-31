package stream

import (
	"context"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

var ErrQueueFull = errors.New("frame queue is full")

// FrameProcessor is a bounded, concurrent pipeline for already-authorized
// frames. It deliberately does not sniff interfaces, intercept unrelated
// sockets, or bypass transport encryption.
type FrameProcessor func(context.Context, []byte) error

type PipelineConfig struct {
	Workers int
	Queue   int
	MaxSize int
}

type PipelineStats struct {
	Accepted uint64 `json:"accepted"`
	Dropped  uint64 `json:"dropped"`
	Handled  uint64 `json:"handled"`
	Failed   uint64 `json:"failed"`
}

type Pipeline struct {
	jobs     chan []byte
	fn       FrameProcessor
	max      int
	accepted atomic.Uint64
	dropped  atomic.Uint64
	handled  atomic.Uint64
	failed   atomic.Uint64
	wg       sync.WaitGroup
	once     sync.Once
	submitMu sync.RWMutex
	closed   bool
}

func NewPipeline(cfg PipelineConfig, fn FrameProcessor) (*Pipeline, error) {
	if fn == nil {
		return nil, errors.New("frame processor is required")
	}
	if cfg.Workers <= 0 {
		cfg.Workers = runtime.GOMAXPROCS(0)
	}
	if cfg.Queue <= 0 {
		cfg.Queue = cfg.Workers * 8
	}
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 64 << 10
	}
	p := &Pipeline{jobs: make(chan []byte, cfg.Queue), fn: fn, max: cfg.MaxSize}
	for i := 0; i < cfg.Workers; i++ {
		p.wg.Add(1)
		go p.worker()
	}
	return p, nil
}

func (p *Pipeline) Submit(frame []byte) error {
	p.submitMu.RLock()
	defer p.submitMu.RUnlock()
	if p.closed {
		return errors.New("pipeline is closed")
	}
	if len(frame) == 0 || len(frame) > p.max {
		return errors.New("frame size outside configured limit")
	}
	copyOfFrame := append([]byte(nil), frame...)
	select {
	case p.jobs <- copyOfFrame:
		p.accepted.Add(1)
		return nil
	default:
		p.dropped.Add(1)
		return ErrQueueFull
	}
}

func (p *Pipeline) worker() {
	defer p.wg.Done()
	for frame := range p.jobs {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := p.fn(ctx, frame)
		cancel()
		if err != nil {
			p.failed.Add(1)
		} else {
			p.handled.Add(1)
		}
	}
}

// Close stops accepting work and drains all queued frames before returning.
func (p *Pipeline) Close() {
	p.once.Do(func() { p.submitMu.Lock(); p.closed = true; close(p.jobs); p.submitMu.Unlock(); p.wg.Wait() })
}

func (p *Pipeline) Stats() PipelineStats {
	return PipelineStats{Accepted: p.accepted.Load(), Dropped: p.dropped.Load(), Handled: p.handled.Load(), Failed: p.failed.Load()}
}
