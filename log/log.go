package log

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
)

func defaultArgs(prefix string, skip int) string {
	_, filePath, line, ok := runtime.Caller(skip)
	if !ok {
		Error("for execute Info!")
		skip -= 1
		return defaultArgs(prefix, skip)
	}

	file := filepath.Base(filePath)
	lineStr := strconv.Itoa(line)
	return prefix + " \033[35m" + file + ":" + lineStr + ":\033[0m"
}

func print(args ...any) (int, error) {
	w := os.Stdout
	var buf bytes.Buffer
	for i, arg := range args {
		if i > 0 {
			buf.WriteByte(' ')
		}
		_, err := fmt.Fprint(&buf, arg)
		if err != nil {
			return 0, err
		}
	}

	if buf.Len() > 0 {
		buf.WriteByte('\n')
		return w.Write(buf.Bytes())
	}

	return 0, nil
}

func printf(value string, args ...any) (int, error) {
	w := os.Stdout
	form := fmt.Sprintf(value, args...)
	return w.Write([]byte(form))
}

func Info(args ...any) {
	print(append([]any{defaultArgs("\033[34m[Info]", 2)}, args...)...)
}

func InfoSkip(skip int, args ...any) {
	print(append([]any{defaultArgs("\033[34m[Info]", skip+2)}, args...)...)
}

func Error(args ...any) {
	print(append([]any{defaultArgs("\033[31m[Error]", 2)}, args...)...)
}

func ErrorSkip(skip int, args ...any) {
	print(append([]any{defaultArgs("\033[31m[Error]", skip+2)}, args...)...)
}

func Warn(args ...any) {
	print(append([]any{defaultArgs("\033[33m[Warn]", 2)}, args...)...)
}

func WarnSkip(skip int, args ...any) {
	print(append([]any{defaultArgs("\033[33m[Warn]", skip+2)}, args...)...)
}

var DebugMode = false

func Debug(args ...any) {
	if DebugMode {
		print(append([]any{defaultArgs("\033[94m[Debug]", 2)}, args...)...)
	}
}

func DebugSkip(skip int, args ...any) {
	if DebugMode {
		print(append([]any{defaultArgs("\033[94m[Debug]", skip+2)}, args...)...)
	}
}
