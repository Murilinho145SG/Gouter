/*
Package log provides colored console logging with file/line information.

Features:
- Colored output for different log levels
- Automatic caller file/line detection
- Customizable call depth tracking
- Simple interface similar to standard log package
*/
package log

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

// ANSI color escape codes for different log levels
const (
	infoColor  = "\033[34m"  // Blue
	warnColor  = "\033[33m"  // Yellow
	errorColor = "\033[31m"  // Red
	debugColor = "\033[35m"  // Magenta
)

// Print handles low-level message formatting and output
// Args:
// - args: Variadic arguments to log
// Returns bytes written and any error
func Print(args ...any) (int, error) {
	w := os.Stdout
	var buf bytes.Buffer
	
	// Format arguments with spaces between them
	for i, arg := range args {
		if i > 0 {
			buf.WriteByte(' ')
		}
		_, err := fmt.Fprint(&buf, arg)
		if err != nil {
			return 0, err
		}
	}

	// Add newline if message not empty
	if buf.Len() > 0 {
		buf.WriteByte('\n')
		return w.Write(buf.Bytes())
	}

	return 0, nil
}

// parserColor formats colored log messages with caller information
// Args:
// - color: ANSI color code
// - prefix: Log level prefix
// - encapsulation: Call stack depth adjustment
// - args: Log message arguments
func parserColor(color, prefix string, encapsulation int, args ...any) {
	// Get caller file and line information
	_, filePath, line, ok := runtime.Caller(encapsulation)
	if !ok {
		Error("failed to get caller info in parserColor!")
	}

	// Format file and line information
	file := filepath.Base(filePath)
	lineStr := strconv.Itoa(line)

	// Construct colored message components
	msg := append([]any{fmt.Sprintf("%s[%s] \033[35m%s:%s\033[0m", 
		color, 
		prefix, 
		file, 
		lineStr)}, 
		args...)
	Print(msg...)
}

// Basic log functions (depth = 2)

// Info logs informational messages (blue)
func Info(args ...any) {
	parserColor(infoColor, "Info", 2, args...)
}

// Warn logs warning messages (yellow)
func Warn(args ...any) {
	parserColor(warnColor, "Warn", 2, args...)
}

// Error logs error messages (red)
func Error(args ...any) {
	parserColor(errorColor, "Error", 2, args...)
}

// Debug logs debug messages (magenta)
func Debug(args ...any) {
	parserColor(debugColor, "Debug", 2, args...)
}

// Extended log functions with custom call depth

// InfoE logs info with custom call depth
func InfoE(encapsulation int, args ...any) {
	parserColor(infoColor, "Info", encapsulation, args...)
}

// WarnE logs warnings with custom call depth
func WarnE(encapsulation int, args ...any) {
	parserColor(warnColor, "Warn", encapsulation, args...)
}

// ErrorE logs errors with custom call depth 
func ErrorE(encapsulation int, args ...any) {
	parserColor(errorColor, "Error", encapsulation, args...)
}

// DebugE logs debug messages with custom call depth
func DebugE(encapsulation int, args ...any) {
	parserColor(debugColor, "Debug", encapsulation, args...)
}