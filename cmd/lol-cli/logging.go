package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

// setupLogging redirects the standard logger to a file. On Windows it writes to
// the executable's directory so users can find it easily; elsewhere it defaults
// to a temp file unless LOL_CLI_LOG is set.
func setupLogging() (cleanup func(), err error) {
	path := logPath()
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log file %s: %w", path, err)
	}

	log.SetOutput(io.MultiWriter(f, os.Stderr))
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("lol-cli started (log: %s)", path)

	return func() {
		log.Printf("lol-cli exiting")
		f.Close()
	}, nil
}

func logPath() string {
	if env := os.Getenv("LOL_CLI_LOG"); env != "" {
		return env
	}
	name := "lol-cli.log"
	if runtime.GOOS == "windows" {
		exe, err := os.Executable()
		if err == nil {
			path := filepath.Join(filepath.Dir(exe), name)
			if isWritable(path) {
				return path
			}
		}
	}
	return filepath.Join(os.TempDir(), name)
}

func isWritable(path string) bool {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return false
	}
	f.Close()
	return true
}

// logPanic recovers from a panic, logs it, and re-raises it.
func logPanic() {
	if r := recover(); r != nil {
		log.Printf("PANIC: %v", r)
		panic(r)
	}
}

// fatal logs the error and exits.
func fatal(format string, args ...any) {
	log.Printf("FATAL: "+format, args...)
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
