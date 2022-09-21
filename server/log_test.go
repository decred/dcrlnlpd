package server

import (
	"sync"
	"testing"

	"github.com/decred/slog"
)

type backend struct {
	mtx  sync.Mutex
	tb   testing.TB
	done bool
}

func (tb *backend) Write(b []byte) (int, error) {
	tb.mtx.Lock()
	if !tb.done {
		tb.tb.Log(string(b[:len(b)-1]))
	}
	tb.mtx.Unlock()
	return len(b), nil
}

func newTestLogBackend(t testing.TB) *backend {
	tb := &backend{tb: t}
	t.Cleanup(func() {
		tb.mtx.Lock()
		tb.done = true
		tb.mtx.Unlock()
	})
	return tb
}

func testLoggerSys(t testing.TB, sys string) slog.Logger {
	bknd := slog.NewBackend(newTestLogBackend(t))
	logg := bknd.Logger(sys)
	logg.SetLevel(slog.LevelTrace)
	return logg
}
