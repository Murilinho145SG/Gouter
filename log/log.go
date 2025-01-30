package log

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

// defaultArgs constructs a formatted log prefix with file name, line number, and a given prefix.
// It uses runtime.Caller to determine the file and line number of the log call.
// The `skip` parameter controls how many stack frames to skip to find the caller.
func defaultArgs(prefix string, skip int) string {
	_, filePath, line, ok := runtime.Caller(skip)
	if !ok {
		// If runtime.Caller fails, log an error and retry with a decremented skip value
		Error("for execute Info!")
		skip -= 1
		return defaultArgs(prefix, skip)
	}

	// Extract the base file name and convert the line number to a string
	file := filepath.Base(filePath)
	lineStr := strconv.Itoa(line)

	// Return the formatted prefix with file name and line number
	return prefix + " \033[35m" + file + ":" + lineStr + ":\033[0m"
}

// print writes the provided arguments to os.Stdout, separated by spaces and followed by a newline.
// It returns the number of bytes written and any error encountered.
func print(args ...any) (int, error) {
	w := os.Stdout
	var buf bytes.Buffer

	// Iterate over the arguments and write them to the buffer
	for i, arg := range args {
		if i > 0 {
			buf.WriteByte(' ') // Add a space between arguments
		}
		_, err := fmt.Fprint(&buf, arg)
		if err != nil {
			return 0, err
		}
	}

	// Add a newline if the buffer is not empty
	if buf.Len() > 0 {
		buf.WriteByte('\n')
		return w.Write(buf.Bytes())
	}

	return 0, nil
}

// printf formats the provided value and arguments using fmt.Sprintf and writes the result to os.Stdout.
// It returns the number of bytes written and any error encountered.
func printf(value string, args ...any) (int, error) {
	w := os.Stdout
	form := fmt.Sprintf(value, args...) // Format the string with arguments
	return w.Write([]byte(form))        // Write the formatted string to stdout
}

// Info logs informational messages with a blue "[Info]" prefix.
// It includes the file name and line number where the log call was made.
func Info(args ...any) {
	print(append([]any{defaultArgs("\033[34m[Info]", 2)}, args...)...)
}

// InfoSkip logs informational messages with a blue "[Info]" prefix, allowing the caller to specify the skip value.
// This is useful for logging from helper functions or wrappers.
func InfoSkip(skip int, args ...any) {
	print(append([]any{defaultArgs("\033[34m[Info]", skip+2)}, args...)...)
}

// Error logs error messages with a red "[Error]" prefix.
// It includes the file name and line number where the log call was made.
func Error(args ...any) {
	print(append([]any{defaultArgs("\033[31m[Error]", 2)}, args...)...)
}

// ErrorSkip logs error messages with a red "[Error]" prefix, allowing the caller to specify the skip value.
// This is useful for logging from helper functions or wrappers.
func ErrorSkip(skip int, args ...any) {
	print(append([]any{defaultArgs("\033[31m[Error]", skip+2)}, args...)...)
}

// Warn logs warning messages with a yellow "[Warn]" prefix.
// It includes the file name and line number where the log call was made.
func Warn(args ...any) {
	print(append([]any{defaultArgs("\033[33m[Warn]", 2)}, args...)...)
}

// WarnSkip logs warning messages with a yellow "[Warn]" prefix, allowing the caller to specify the skip value.
// This is useful for logging from helper functions or wrappers.
func WarnSkip(skip int, args ...any) {
	print(append([]any{defaultArgs("\033[33m[Warn]", skip+2)}, args...)...)
}

// DebugMode controls whether debug logs are printed.
// When set to false, debug logs are ignored.
var DebugMode = false

// Debug logs debug messages with a light blue "[Debug]" prefix, but only if DebugMode is true.
// It includes the file name and line number where the log call was made.
func Debug(args ...any) {
	if DebugMode {
		print(append([]any{defaultArgs("\033[94m[Debug]", 2)}, args...)...)
	}
}

// DebugSkip logs debug messages with a light blue "[Debug]" prefix, allowing the caller to specify the skip value.
// This is useful for logging from helper functions or wrappers.
// Debug logs are only printed if DebugMode is true.
func DebugSkip(skip int, args ...any) {
	if DebugMode {
		print(append([]any{defaultArgs("\033[94m[Debug]", skip+2)}, args...)...)
	}
}