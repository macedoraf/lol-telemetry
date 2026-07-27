package recorder

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"
)

const (
	chanCapacity = 1024
	writerBuf    = 32 * 1024
	flushEvery   = 5 * time.Second
)

// Recorder persists raw telemetry snapshots and judge tips to disk as
// append-only JSONL. It is safe for concurrent use: Record/RecordTip never
// block the caller.
type Recorder struct {
	dir       string
	ch        chan TelemetryRecord
	tipCh     chan TipRecord
	stop      chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup
	sessionMu sync.Mutex
	session   *sessionWriter
	tipFile   *os.File
	tipWriter *bufio.Writer

	written     int64
	dropped     int64
	tipWritten  int64
	tipDropped  int64
}

// sessionWriter holds the currently open file for the active session.
type sessionWriter struct {
	id     string
	file   *os.File
	writer *bufio.Writer
}

// New creates the recordings directory and starts the background writer.
func New(dir string) (*Recorder, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create recordings dir: %w", err)
	}

	r := &Recorder{
		dir:   dir,
		ch:    make(chan TelemetryRecord, chanCapacity),
		tipCh: make(chan TipRecord, chanCapacity),
		stop:  make(chan struct{}),
	}
	r.wg.Add(1)
	go r.loop()
	return r, nil
}

// StartSession opens a new JSONL file under <dir>/<id>/telemetry.jsonl.
func (r *Recorder) StartSession(id string) error {
	r.sessionMu.Lock()
	defer r.sessionMu.Unlock()

	if r.session != nil {
		// Close previous session gracefully before starting a new one.
		r.closeSessionLocked()
	}

	sessionDir := filepath.Join(r.dir, id)
	if err := os.MkdirAll(sessionDir, 0o755); err != nil {
		return fmt.Errorf("create session dir: %w", err)
	}

	path := filepath.Join(sessionDir, "telemetry.jsonl")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("open telemetry.jsonl: %w", err)
	}

	r.session = &sessionWriter{
		id:     id,
		file:   f,
		writer: bufio.NewWriterSize(f, writerBuf),
	}
	return nil
}

// EndSession flushes and closes the current session file.
func (r *Recorder) EndSession() {
	r.sessionMu.Lock()
	defer r.sessionMu.Unlock()
	r.closeSessionLocked()
}

// Record enqueues a record for async writing. If the channel is full the
// record is dropped and a counter is incremented. This is intentional: the
// poll loop must never block on disk I/O.
func (r *Recorder) Record(rec TelemetryRecord) {
	select {
	case r.ch <- rec:
	default:
		d := atomic.AddInt64(&r.dropped, 1)
		if d%100 == 0 {
			log.Printf("recorder: dropped %d telemetry records (backpressure)", d)
		}
	}
}

// SessionID returns the active session id or "" if none is open.
func (r *Recorder) SessionID() string {
	r.sessionMu.Lock()
	defer r.sessionMu.Unlock()
	if r.session == nil {
		return ""
	}
	return r.session.id
}

// Written returns the number of records successfully written to disk.
func (r *Recorder) Written() int64 {
	return atomic.LoadInt64(&r.written)
}

// Dropped returns the number of records dropped due to channel backpressure.
func (r *Recorder) Dropped() int64 {
	return atomic.LoadInt64(&r.dropped)
}

// Close stops the writer, drains pending records, flushes the current session
// and waits for the goroutine to exit. It honors ctx timeout. Close is safe to
// call more than once.
func (r *Recorder) Close(ctx context.Context) error {
	r.closeOnce.Do(func() { close(r.stop) })

	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

func (r *Recorder) loop() {
	defer r.wg.Done()
	flushTicker := time.NewTicker(flushEvery)
	defer flushTicker.Stop()

	for {
		select {
		case <-r.stop:
			r.drainAndFlush()
			return
		case rec := <-r.ch:
			r.write(rec)
		case tip := <-r.tipCh:
			r.writeTip(tip)
		case <-flushTicker.C:
			r.sessionMu.Lock()
			if r.tipWriter != nil {
				_ = r.tipWriter.Flush()
			}
			if r.session != nil {
				_ = r.session.writer.Flush()
			}
			r.sessionMu.Unlock()
		}
	}
}

// drainAndFlush consumes any remaining records in the channel and flushes.
func (r *Recorder) drainAndFlush() {
	for {
		select {
		case rec := <-r.ch:
			r.write(rec)
		case tip := <-r.tipCh:
			r.writeTip(tip)
		default:
			r.sessionMu.Lock()
			if r.tipWriter != nil {
				_ = r.tipWriter.Flush()
			}
			if r.session != nil {
				_ = r.session.writer.Flush()
			}
			r.closeSessionLocked()
			r.sessionMu.Unlock()
			return
		}
	}
}

func (r *Recorder) write(rec TelemetryRecord) {
	r.sessionMu.Lock()
	defer r.sessionMu.Unlock()

	if r.session == nil {
		// No active session; drop silently. This should not happen in normal
		// operation because StartSession is called before Record.
		return
	}

	line, err := json.Marshal(rec)
	if err != nil {
		log.Printf("recorder: failed to marshal telemetry record: %v", err)
		return
	}

	w := r.session.writer
	if _, err := w.Write(line); err != nil {
		log.Printf("recorder: write error: %v", err)
		return
	}
	if err := w.WriteByte('\n'); err != nil {
		log.Printf("recorder: write newline error: %v", err)
		return
	}
	atomic.AddInt64(&r.written, 1)
}

func (r *Recorder) closeSessionLocked() {
	if r.session == nil {
		return
	}
	_ = r.session.writer.Flush()
	_ = r.session.file.Close()
	// Close tips writer tied to this session.
	if r.tipWriter != nil {
		_ = r.tipWriter.Flush()
	}
	if r.tipFile != nil {
		_ = r.tipFile.Close()
	}
	r.session = nil
	r.tipWriter = nil
	r.tipFile = nil
}
