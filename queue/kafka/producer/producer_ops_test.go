package producer

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/segmentio/kafka-go"
)

type fakeWriter struct {
	writeErr error
	closeErr error
	msgs     []kafka.Message
	closed   bool
}

func (f *fakeWriter) WriteMessages(_ context.Context, msgs ...kafka.Message) error {
	f.msgs = append(f.msgs, msgs...)
	return f.writeErr
}

func (f *fakeWriter) Close() error {
	f.closed = true
	return f.closeErr
}

func code(t *testing.T, err error) stderr.Code {
	t.Helper()
	var se *stderr.Error
	if !errors.As(err, &se) {
		t.Fatalf("expected *stderr.Error, got %T (%v)", err, err)
	}
	return se.Code
}

func TestPublish(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	boom := errors.New("broker down")

	tests := []struct {
		name     string
		ctx      context.Context
		err      error
		wantCode stderr.Code
	}{
		{"success", context.Background(), nil, 0},
		{"canceled maps to timeout", canceled, boom, stderr.CodeTimeout},
		{"broker error maps to unavailable", context.Background(), boom, stderr.CodeUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &fakeWriter{writeErr: tt.err}
			p := &impl{writer: w, topic: "t"}
			err := p.Publish(tt.ctx, []byte("k"), []byte("v"))
			if tt.wantCode == 0 {
				if err != nil {
					t.Fatal(err)
				}
			} else if code(t, err) != tt.wantCode {
				t.Fatalf("code = %v, want %v", code(t, err), tt.wantCode)
			}
			if len(w.msgs) != 1 || string(w.msgs[0].Key) != "k" || string(w.msgs[0].Value) != "v" {
				t.Errorf("message not forwarded: %+v", w.msgs)
			}
		})
	}
}

func TestPublishJSON(t *testing.T) {
	t.Run("serializes payload and key", func(t *testing.T) {
		w := &fakeWriter{}
		p := &impl{writer: w}
		if err := p.PublishJSON(context.Background(), "user-1", map[string]int{"n": 3}); err != nil {
			t.Fatal(err)
		}
		var got map[string]int
		if err := json.Unmarshal(w.msgs[0].Value, &got); err != nil || got["n"] != 3 || string(w.msgs[0].Key) != "user-1" {
			t.Errorf("unexpected message %+v (%v)", w.msgs[0], err)
		}
	})

	t.Run("unmarshalable payload is a bad request and nothing is written", func(t *testing.T) {
		w := &fakeWriter{}
		err := (&impl{writer: w}).PublishJSON(context.Background(), "k", make(chan int))
		if code(t, err) != stderr.CodeBadRequest || len(w.msgs) != 0 {
			t.Errorf("code=%v written=%d", code(t, err), len(w.msgs))
		}
	})

	t.Run("writer failure propagates", func(t *testing.T) {
		err := (&impl{writer: &fakeWriter{writeErr: errors.New("x")}}).PublishJSON(context.Background(), "k", 1)
		if code(t, err) != stderr.CodeUnavailable {
			t.Errorf("code = %v", code(t, err))
		}
	})
}

func TestClose(t *testing.T) {
	w := &fakeWriter{}
	if err := (&impl{writer: w}).Close(); err != nil || !w.closed {
		t.Fatalf("err=%v closed=%v", err, w.closed)
	}
	if code(t, (&impl{writer: &fakeWriter{closeErr: errors.New("x")}}).Close()) != stderr.CodeInternal {
		t.Error("close failure must map to CodeInternal")
	}
}

func TestOptionsAndDefaults(t *testing.T) {
	cfg := Config{Brokers: []string{"localhost:9092"}, Topic: "t"}

	p, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	w := p.(*impl).writer.(*kafka.Writer)
	if _, ok := w.Balancer.(*kafka.LeastBytes); !ok || w.Async {
		t.Errorf("defaults: balancer=%T async=%v", w.Balancer, w.Async)
	}
	_ = p.Close()

	p, err = New(cfg, WithBalancer(&kafka.Hash{}), WithAsync(true), WithBalancer(nil))
	if err != nil {
		t.Fatal(err)
	}
	w = p.(*impl).writer.(*kafka.Writer)
	if _, ok := w.Balancer.(*kafka.Hash); !ok || !w.Async {
		t.Errorf("options not applied (nil balancer must be ignored): balancer=%T async=%v", w.Balancer, w.Async)
	}
	_ = p.Close()

	tests := []struct {
		name string
		in   Config
		want Config
	}{
		{"zero values get defaults", Config{}, DefaultConfig()},
		{"explicit values kept", Config{BatchSize: 5, BatchTimeout: time.Second, RequiredAcks: 1},
			Config{BatchSize: 5, BatchTimeout: time.Second, RequiredAcks: 1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in
			got.applyDefaults()
			if got.BatchSize != tt.want.BatchSize || got.BatchTimeout != tt.want.BatchTimeout || got.RequiredAcks != tt.want.RequiredAcks {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
