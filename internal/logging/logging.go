package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
)

var logger *log.Logger

// Init opens the application log file. dir is e.g. %APPDATA%\DesktopPet\logs.
// Writes are mirrored to stderr so debug builds show output in the console.
func Init(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(dir, "app.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	logger = log.New(io.MultiWriter(f, os.Stderr), "", log.LstdFlags)
	return nil
}

// Printf logs an important event. Must stay cheap and never be called per-frame.
func Printf(format string, args ...interface{}) {
	if logger == nil {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
		return
	}
	logger.Printf(format, args...)
}
