package logbuf_test

import (
	"sync"
	"testing"
	"time"

	"github.com/user/vpm/internal/logbuf"
)

func TestBuffer_FlushesWithin200ms(t *testing.T) {
	var mu sync.Mutex
	var received []string

	buf := logbuf.New("srv1", func(serverID string, lines []string) {
		mu.Lock()
		received = append(received, lines...)
		mu.Unlock()
	})
	defer buf.Close()

	buf.Write([]byte("line 1\n"))
	buf.Write([]byte("line 2\n"))

	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if len(received) < 2 {
		t.Errorf("expected 2 lines flushed within 200ms, got %d", len(received))
	}
}

func TestBuffer_CloseFlushesRemaining(t *testing.T) {
	var mu sync.Mutex
	var received []string

	buf := logbuf.New("srv2", func(_ string, lines []string) {
		mu.Lock()
		received = append(received, lines...)
		mu.Unlock()
	})

	buf.Write([]byte("final line\n"))
	buf.Close()

	mu.Lock()
	defer mu.Unlock()
	if len(received) == 0 {
		t.Error("Close() should flush remaining lines")
	}
}

func TestBuffer_NoFlushOnEmpty(t *testing.T) {
	calls := 0
	buf := logbuf.New("srv3", func(_ string, lines []string) {
		calls++
	})
	time.Sleep(200 * time.Millisecond)
	buf.Close()
	if calls != 0 {
		t.Errorf("expected 0 flush calls on empty buffer, got %d", calls)
	}
}
