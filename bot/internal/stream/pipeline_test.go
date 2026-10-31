package stream

import (
	"context"
	"sync/atomic"
	"testing"
)

func TestPipelineProcessesSubmittedFrames(t *testing.T) {
	var n atomic.Int64
	p, err := NewPipeline(PipelineConfig{Workers: 2, Queue: 8, MaxSize: 128}, func(context.Context, []byte) error { n.Add(1); return nil })
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		if err := p.Submit([]byte(`{"event":"test"}`)); err != nil {
			t.Fatal(err)
		}
	}
	p.Close()
	if got := n.Load(); got != 5 {
		t.Fatalf("processed %d frames, want 5", got)
	}
	if got := p.Stats().Handled; got != 5 {
		t.Fatalf("handled=%d, want 5", got)
	}
}

func TestPipelineRejectsOversize(t *testing.T) {
	p, err := NewPipeline(PipelineConfig{Workers: 1, Queue: 1, MaxSize: 2}, func(context.Context, []byte) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	if err := p.Submit([]byte("123")); err == nil {
		t.Fatal("expected size error")
	}
}
