//ff:what 100ms 슬라이딩 배치 버퍼 → Wails events 방출
//ff:why 고빈도 stdout 라인을 하나씩 IPC로 보내면 UI 프리징 → 배치로 병합
package logbuf

import (
	"sync"
	"time"
)

const flushInterval = 100 * time.Millisecond

// Flush is the callback type invoked by Buffer when lines are ready.
// serverID identifies the server whose log lines are being flushed.
type Flush func(serverID string, lines []string)

// Buffer accumulates log lines and flushes them in 100ms batches.
type Buffer struct {
	mu       sync.Mutex
	serverID string
	lines    []string
	flush    Flush
	ticker   *time.Ticker
	stop     chan struct{}
}

// New creates and starts a Buffer for the given server.
// Call Close() when the server stops.
func New(serverID string, flush Flush) *Buffer {
	b := &Buffer{
		serverID: serverID,
		flush:    flush,
		ticker:   time.NewTicker(flushInterval),
		stop:     make(chan struct{}),
	}
	go b.run()
	return b
}

// Write implements io.Writer; appends lines to the batch.
func (b *Buffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	b.lines = append(b.lines, string(p))
	b.mu.Unlock()
	return len(p), nil
}

// Close flushes remaining lines and stops the ticker.
func (b *Buffer) Close() {
	close(b.stop)
	b.ticker.Stop()
	b.drain()
}

func (b *Buffer) run() {
	for {
		select {
		case <-b.ticker.C:
			b.drain()
		case <-b.stop:
			return
		}
	}
}

func (b *Buffer) drain() {
	b.mu.Lock()
	if len(b.lines) == 0 {
		b.mu.Unlock()
		return
	}
	batch := make([]string, len(b.lines))
	copy(batch, b.lines)
	b.lines = b.lines[:0]
	b.mu.Unlock()
	b.flush(b.serverID, batch)
}
