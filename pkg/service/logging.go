package service

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
)

func SetupLogging(appName, envVar string) (cleanup func(), err error) {
	path := logPath(appName, envVar)
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log file %s: %w", path, err)
	}

	log.SetOutput(io.MultiWriter(f, os.Stderr))
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Printf("%s started (log: %s)", appName, path)

	return func() {
		log.Printf("%s exiting", appName)
		f.Close()
	}, nil
}

func logPath(appName, envVar string) string {
	if env := os.Getenv(envVar); env != "" {
		return env
	}
	name := appName + ".log"
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

func LogPanic() {
	if r := recover(); r != nil {
		log.Printf("PANIC: %v", r)
		panic(r)
	}
}

func Fatal(format string, args ...any) {
	log.Printf("FATAL: "+format, args...)
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
