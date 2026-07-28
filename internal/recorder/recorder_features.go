package recorder

import (
	"bufio"
	"encoding/json"
	"log"
	"os"
	"path/filepath"
	"sync/atomic"
)

// RecordFeature enqueues a feature vector for async writing to features.jsonl
// in the current session directory. Like Record, it never blocks and drops on
// backpressure.
func (r *Recorder) RecordFeature(rec FeatureRecord) {
	select {
	case r.featureCh <- rec:
	default:
		d := atomic.AddInt64(&r.featureDropped, 1)
		if d%100 == 0 {
			log.Printf("recorder: dropped %d feature records (backpressure)", d)
		}
	}
}

// FeatureWritten returns the number of feature vectors successfully written to disk.
func (r *Recorder) FeatureWritten() int64 {
	return atomic.LoadInt64(&r.featureWritten)
}

// FeatureDropped returns the number of feature vectors dropped due to backpressure.
func (r *Recorder) FeatureDropped() int64 {
	return atomic.LoadInt64(&r.featureDropped)
}

// ensureFeaturesWriter opens features.jsonl lazily inside the current session dir.
func (r *Recorder) ensureFeaturesWriter() (*bufio.Writer, *os.File, error) {
	if r.featureFile != nil {
		return r.featureWriter, r.featureFile, nil
	}
	if r.session == nil {
		return nil, nil, errNoSession
	}
	path := filepath.Join(r.dir, r.session.id, "features.jsonl")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, nil, err
	}
	r.featureFile = f
	r.featureWriter = bufio.NewWriterSize(f, writerBuf)
	return r.featureWriter, f, nil
}

func (r *Recorder) writeFeature(rec FeatureRecord) {
	r.sessionMu.Lock()
	defer r.sessionMu.Unlock()

	if r.session == nil {
		return
	}
	rec.Session = r.session.id

	w, _, err := r.ensureFeaturesWriter()
	if err != nil {
		log.Printf("recorder: failed to open features writer: %v", err)
		return
	}

	line, err := json.Marshal(rec)
	if err != nil {
		log.Printf("recorder: failed to marshal feature record: %v", err)
		return
	}
	if _, err := w.Write(line); err != nil {
		log.Printf("recorder: feature write error: %v", err)
		return
	}
	if err := w.WriteByte('\n'); err != nil {
		log.Printf("recorder: feature write newline error: %v", err)
		return
	}
	atomic.AddInt64(&r.featureWritten, 1)
}

// closeFeatures closes the features writer and resets it for the next session.
// Caller must hold sessionMu.
func (r *Recorder) closeFeatures() {
	if r.featureWriter != nil {
		_ = r.featureWriter.Flush()
	}
	if r.featureFile != nil {
		_ = r.featureFile.Close()
	}
	r.featureWriter = nil
	r.featureFile = nil
}
