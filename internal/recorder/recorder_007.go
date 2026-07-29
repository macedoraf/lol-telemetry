package recorder

import (
	"bufio"
	"encoding/json"
	"errors"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
)

var errNoSession = errors.New("no active session")

// RecordTip enqueues a Judge tip for async writing to tips.jsonl in the
// current session directory. Like Record, it never blocks and drops on
// backpressure.
func (r *Recorder) RecordTip(tip TipRecord) {
	select {
	case r.tipCh <- tip:
	default:
		d := atomic.AddInt64(&r.tipDropped, 1)
		if d%100 == 0 {
			log.Printf("recorder: dropped %d tip records (backpressure)", d)
		}
	}
}

// TipWritten returns the number of tips successfully written to disk.
func (r *Recorder) TipWritten() int64 {
	return atomic.LoadInt64(&r.tipWritten)
}

// TipDropped returns the number of tips dropped due to channel backpressure.
func (r *Recorder) TipDropped() int64 {
	return atomic.LoadInt64(&r.tipDropped)
}

// ensureTipsWriter opens tips.jsonl lazily inside the current session dir.
func (r *Recorder) ensureTipsWriter() (*bufio.Writer, *os.File, error) {
	if r.tipFile != nil {
		return r.tipWriter, r.tipFile, nil
	}
	if r.session == nil {
		return nil, nil, errNoSession
	}
	path := filepath.Join(r.dir, r.session.id, "tips.jsonl")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, nil, err
	}
	r.tipFile = f
	r.tipWriter = bufio.NewWriterSize(f, writerBuf)
	return r.tipWriter, f, nil
}

func (r *Recorder) writeTip(tip TipRecord) {
	r.sessionMu.Lock()
	defer r.sessionMu.Unlock()

	if r.session == nil {
		return
	}
	tip.Session = r.session.id

	w, _, err := r.ensureTipsWriter()
	if err != nil {
		log.Printf("recorder: failed to open tips writer: %v", err)
		return
	}

	line, err := json.Marshal(tip)
	if err != nil {
		log.Printf("recorder: failed to marshal tip: %v", err)
		return
	}
	if _, err := w.Write(line); err != nil {
		log.Printf("recorder: tip write error: %v", err)
		return
	}
	if err := w.WriteByte('\n'); err != nil {
		log.Printf("recorder: tip write newline error: %v", err)
		return
	}
	atomic.AddInt64(&r.tipWritten, 1)
}

// closeTips closes the tips writer and resets it for the next session.
// Caller must hold sessionMu.
func (r *Recorder) closeTips() {
	if r.tipWriter != nil {
		_ = r.tipWriter.Flush()
	}
	if r.tipFile != nil {
		_ = r.tipFile.Close()
	}
	r.tipWriter = nil
	r.tipFile = nil
}
