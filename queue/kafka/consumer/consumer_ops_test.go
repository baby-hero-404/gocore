package consumer

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/segmentio/kafka-go"
)

type fetchResult struct {
	msg kafka.Message
	err error
}

// fakeReader replays scripted fetch results, then io.EOF.
type fakeReader struct {
	fetches   []fetchResult
	commitErr error
	closeErr  error
	committed []kafka.Message
	closed    bool
	onFetch   func(call int)
	calls     int
}

func (f *fakeReader) FetchMessage(context.Context) (kafka.Message, error) {
	f.calls++
	if f.onFetch != nil {
		f.onFetch(f.calls)
	}
	if len(f.fetches) == 0 {
		return kafka.Message{}, io.EOF
	}
	r := f.fetches[0]
	f.fetches = f.fetches[1:]
	return r.msg, r.err
}

func (f *fakeReader) CommitMessages(_ context.Context, msgs ...kafka.Message) error {
	f.committed = append(f.committed, msgs...)
	return f.commitErr
}

func (f *fakeReader) Close() error {
	f.closed = true
	return f.closeErr
}

func newFake(r *fakeReader) *impl { return &impl{reader: r, topic: "t"} }

func code(t *testing.T, err error) stderr.Code {
	t.Helper()
	var se *stderr.Error
	if !errors.As(err, &se) {
		t.Fatalf("expected *stderr.Error, got %T (%v)", err, err)
	}
	return se.Code
}

func msg(offset int64, value string) kafka.Message {
	return kafka.Message{Topic: "t", Partition: 2, Offset: offset, Key: []byte("k"), Value: []byte(value),
		Headers: []kafka.Header{{Key: "h", Value: []byte("v")}}, Time: time.Unix(100, 0)}
}

func TestFetchMessage(t *testing.T) {
	canceled, cancel := context.WithCancel(context.Background())
	cancel()
	expired, cancel2 := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel2()
	boom := errors.New("broker down")

	tests := []struct {
		name     string
		ctx      context.Context
		result   fetchResult
		wantErr  error // passed through unchanged
		wantCode stderr.Code
	}{
		{"success", context.Background(), fetchResult{msg: msg(7, "x")}, nil, 0},
		{"eof passes through", context.Background(), fetchResult{err: io.EOF}, io.EOF, 0},
		{"canceled passes through", canceled, fetchResult{err: context.Canceled}, context.Canceled, 0},
		{"deadline maps to timeout", expired, fetchResult{err: boom}, nil, stderr.CodeTimeout},
		{"broker error maps to unavailable", context.Background(), fetchResult{err: boom}, nil, stderr.CodeUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := newFake(&fakeReader{fetches: []fetchResult{tt.result}})
			got, err := c.FetchMessage(tt.ctx)
			switch {
			case tt.wantErr != nil:
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
			case tt.wantCode != 0:
				if code(t, err) != tt.wantCode {
					t.Fatalf("code = %v, want %v", code(t, err), tt.wantCode)
				}
			default:
				if err != nil {
					t.Fatal(err)
				}
				if got.Topic != "t" || got.Partition != 2 || got.Offset != 7 || string(got.Value) != "x" || got.Headers["h"] != "v" || got.Raw.Offset != 7 {
					t.Errorf("message not mapped: %+v", got)
				}
			}
		})
	}
}

func TestCommitMessages(t *testing.T) {
	expired, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()
	boom := errors.New("coordinator gone")

	tests := []struct {
		name     string
		ctx      context.Context
		err      error
		wantCode stderr.Code
	}{
		{"success", context.Background(), nil, 0},
		{"timeout", expired, boom, stderr.CodeTimeout},
		{"unavailable", context.Background(), boom, stderr.CodeUnavailable},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeReader{commitErr: tt.err}
			err := newFake(f).CommitMessages(tt.ctx, Message{Raw: msg(1, "a")}, Message{Raw: msg(2, "b")})
			if tt.wantCode == 0 {
				if err != nil {
					t.Fatal(err)
				}
			} else if code(t, err) != tt.wantCode {
				t.Fatalf("code = %v, want %v", code(t, err), tt.wantCode)
			}
			if len(f.committed) != 2 || f.committed[0].Offset != 1 || f.committed[1].Offset != 2 {
				t.Errorf("raw messages not forwarded: %+v", f.committed)
			}
		})
	}
}

func TestClose(t *testing.T) {
	f := &fakeReader{}
	if err := newFake(f).Close(); err != nil || !f.closed {
		t.Fatalf("close: err=%v closed=%v", err, f.closed)
	}
	err := newFake(&fakeReader{closeErr: errors.New("x")}).Close()
	if code(t, err) != stderr.CodeInternal {
		t.Errorf("code = %v, want CodeInternal", code(t, err))
	}
}

func TestSubscribe(t *testing.T) {
	boom := errors.New("handler failed")

	t.Run("commits after successful handler, stops at EOF", func(t *testing.T) {
		f := &fakeReader{fetches: []fetchResult{{msg: msg(1, "a")}, {msg: msg(2, "b")}}}
		var seen []string
		err := newFake(f).Subscribe(context.Background(), func(_ context.Context, m Message) error {
			seen = append(seen, string(m.Value))
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(seen) != 2 || len(f.committed) != 2 {
			t.Errorf("seen=%v committed=%d", seen, len(f.committed))
		}
	})

	t.Run("does not commit when handler fails and keeps consuming", func(t *testing.T) {
		f := &fakeReader{fetches: []fetchResult{{msg: msg(1, "bad")}, {msg: msg(2, "good")}}}
		err := newFake(f).Subscribe(context.Background(), func(_ context.Context, m Message) error {
			if string(m.Value) == "bad" {
				return boom
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(f.committed) != 1 || f.committed[0].Offset != 2 {
			t.Errorf("only the good message should be committed: %+v", f.committed)
		}
	})

	t.Run("commit failure is logged, loop continues", func(t *testing.T) {
		f := &fakeReader{fetches: []fetchResult{{msg: msg(1, "a")}, {msg: msg(2, "b")}}, commitErr: errors.New("commit")}
		n := 0
		err := newFake(f).Subscribe(context.Background(), func(context.Context, Message) error { n++; return nil })
		if err != nil || n != 2 {
			t.Fatalf("err=%v handled=%d", err, n)
		}
	})

	t.Run("fetch error maps to unavailable", func(t *testing.T) {
		f := &fakeReader{fetches: []fetchResult{{err: errors.New("broker down")}}}
		err := newFake(f).Subscribe(context.Background(), func(context.Context, Message) error { return nil })
		if code(t, err) != stderr.CodeUnavailable {
			t.Fatalf("code = %v", code(t, err))
		}
	})

	t.Run("already canceled context returns its error without fetching", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		f := &fakeReader{fetches: []fetchResult{{msg: msg(1, "a")}}}
		err := newFake(f).Subscribe(ctx, func(context.Context, Message) error { return nil })
		if !errors.Is(err, context.Canceled) || f.calls != 0 {
			t.Fatalf("err=%v fetch calls=%d", err, f.calls)
		}
	})

	t.Run("cancellation during fetch ends the loop cleanly", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		f := &fakeReader{fetches: []fetchResult{{err: context.Canceled}}, onFetch: func(int) { cancel() }}
		if err := newFake(f).Subscribe(ctx, func(context.Context, Message) error { return nil }); err != nil {
			t.Fatalf("want nil on cancellation, got %v", err)
		}
	})
}

func TestMessageBindJSON(t *testing.T) {
	type payload struct {
		ID int `json:"id"`
	}
	tests := []struct {
		name    string
		value   string
		wantErr bool
		wantID  int
	}{
		{"valid", `{"id":5}`, false, 5},
		{"invalid json", `{`, true, 0},
		{"empty", ``, true, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Message{Value: []byte(tt.value)}
			var p payload
			err := m.BindJSON(&p)
			if (err != nil) != tt.wantErr || p.ID != tt.wantID {
				t.Errorf("err=%v id=%d", err, p.ID)
			}
		})
	}
}

func TestOptionsAndDefaults(t *testing.T) {
	cfg := Config{Brokers: []string{"localhost:9092"}, Topic: "t", GroupID: "g"}
	c, err := New(cfg, WithStartOffset(-1))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = c.Close() }()
	r := c.(*impl).reader.(*kafka.Reader)
	if r.Config().StartOffset != -1 {
		t.Errorf("start offset = %d, want -1", r.Config().StartOffset)
	}

	tests := []struct {
		name string
		in   Config
		want Config
	}{
		{"zero values get defaults", Config{}, DefaultConfig()},
		{"explicit values kept", Config{MinBytes: 1, MaxBytes: 2, MaxWait: time.Second * 3, StartOffset: -1},
			Config{MinBytes: 1, MaxBytes: 2, MaxWait: time.Second * 3, StartOffset: -1}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.in
			got.applyDefaults()
			if got.MinBytes != tt.want.MinBytes || got.MaxBytes != tt.want.MaxBytes || got.MaxWait != tt.want.MaxWait || got.StartOffset != tt.want.StartOffset {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}
