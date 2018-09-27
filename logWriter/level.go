package logWriter

import (
	"fmt"
	"strings"
)

// Level type
type Level uint32

// Convert the Level to a string. E.g. DebugLevel becomes "debug".
func (level Level) String() string {
	switch level {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warning"
	case ErrorLevel:
		return "error"
	}

	return "unknown"
}

// ParseLevel takes a string level and returns the log level constant.
func ParseLevel(lvl string) (Level, error) {
	switch strings.ToLower(lvl) {
	case "error":
		return ErrorLevel, nil
	case "warn", "warning":
		return WarnLevel, nil
	case "info":
		return InfoLevel, nil
	case "debug":
		return DebugLevel, nil
	}

	var l Level
	return l, fmt.Errorf("not a valid logrus Level: %q", lvl)
}

// A constant exposing all logging levels
var AllLevels = []Level{
	ErrorLevel,
	WarnLevel,
	InfoLevel,
	DebugLevel,
}

// These are the different logging levels.
const (
	// ErrorLevel level. Logs. Used for errors that should definitely be noted.
	ErrorLevel Level = iota
	// WarnLevel level. Non-critical entries that deserve eyes.
	WarnLevel
	// InfoLevel level. General operational entries about what's going on inside the
	// application.
	InfoLevel
	// DebugLevel level. Usually only enabled when debugging. Very verbose logging.
	DebugLevel
)
