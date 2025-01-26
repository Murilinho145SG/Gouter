package httpio

import (
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/Murilinho145SG/gouter/gouter/buffer"
)

type Headers map[string]string

var (
	ErrEOF           = errors.New("EOF")
	ErrNotExist      = errors.New("value not exists")
	ErrAlreadyExists = errors.New("value already exists")
	ErrInvalidHeader = errors.New("invalid header in request")
)

type Request struct {
	Method  string
	Path    string
	Headers Headers
	Version string
	Body    buffer.BuffReader
}

func NewRequest() *Request {
	return &Request{
		Headers: make(Headers),
	}
}

func (r *Request) SetBody(body io.Reader) {
	if body == nil {
		return
	}

	lengthStr, err := r.Headers.Get("Content-Length")
	var length int
	if err == nil {
		var err error
		length, err = strconv.Atoi(strings.TrimSpace(lengthStr))
		if err != nil {
			length = 0
		}
	}

	br := buffer.NewBuffReader(body, length)
	r.Body = br
}

func (r *Request) Parser(headersByte []byte) error {
	raw_headers := string(headersByte)
	lines := strings.Split(raw_headers, "\r\n")
	titleParts := strings.Split(lines[0], " ")
	if len(titleParts) > 0 && len(titleParts) == 3 {
		r.Method = titleParts[0]
		r.Path = titleParts[1]
		r.Version = titleParts[2]
	}

	for i := 1; i < len(lines); i++ {
		line := lines[i]
		parts := strings.SplitN(line, ":", 2)

		if len(parts) == 2 {
			key := parts[0]
			value := parts[1]
			r.Headers.Add(key, value)
		} else {
			return ErrInvalidHeader
		}
	}

	return nil
}

func (h Headers) Add(key, value string) error {
	_, err := h.Get(key)
	if err == nil {
		return ErrAlreadyExists
	}

	h[key] = value

	return nil
}

func (h Headers) Get(key string) (string, error) {
	value := h[key]
	if value == "" {
		return "", ErrNotExist
	}

	return value, nil
}

func (h Headers) Set(key, value string) {
	h[key] = value
}

func (h Headers) Del(key string) error {
	_, err := h.Get(key)
	if err != nil {
		return err
	}

	delete(h, key)

	return nil
}
