package lock

import (
	"context"
	"errors"
	"testing"

	"github.com/baby-hero-404/gocore/stderr"
	"github.com/go-redsync/redsync/v4"
)

func TestClassifyAcquireError(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	bg := context.Background()

	tests := []struct {
		name string
		ctx  context.Context
		err  error
		want stderr.Code
	}{
		{"quorum taken", bg, &redsync.ErrTaken{Nodes: []int{0, 1}}, stderr.CodeLockHeld},
		{"wrapped quorum taken", bg, errors.Join(errors.New("wrap"), &redsync.ErrTaken{Nodes: []int{0}}), stderr.CodeLockHeld},
		{"retries exhausted", bg, redsync.ErrFailed, stderr.CodeLockHeld},
		{"single node taken", bg, &redsync.ErrNodeTaken{Node: 0}, stderr.CodeLockHeld},
		{"redis node error", bg, &redsync.RedisError{Node: 0, Err: errors.New("dial refused")}, stderr.CodeUnavailable},
		{"redis error beats node taken", bg, errors.Join(&redsync.ErrNodeTaken{Node: 1}, &redsync.RedisError{Node: 0, Err: errors.New("down")}), stderr.CodeUnavailable},
		{"unknown error", bg, errors.New("surprise"), stderr.CodeUnavailable},
		{"cancelled context", cancelled, redsync.ErrFailed, stderr.CodeTimeout},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, msg := classifyAcquireError(tt.ctx, tt.err); got != tt.want || msg == "" {
				t.Errorf("code = %v (msg %q), want %v", got, msg, tt.want)
			}
		})
	}
}
