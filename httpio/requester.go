package httpio

import (
	"errors"
	"io"
	"net/textproto"
	"strconv"
	"strings"

	"github.com/Murilinho145SG/gouter/buffer"
	"github.com/Murilinho145SG/gouter/log"
)

// Headers represents a collection of HTTP headers with case-insensitive keys.
type Headers map[string]string

// Predefined errors for request handling.
var (
	// ErrEOF indicates that the end of the input has been reached.
	ErrEOF = errors.New("EOF")

	// ErrNotExist is returned when a requested value does not exist.
	ErrNotExist = errors.New("value does not exist")

	// ErrAlreadyExists is returned when attempting to add a value that already exists.
	ErrAlreadyExists = errors.New("value already exists")

	// ErrInvalidHeader is returned when an invalid header format is encountered.
	ErrInvalidHeader = errors.New("invalid header in request")
)

// Params represents a collection of key-value pairs used for request parameters.
type Params map[string]string

// Request represents an HTTP request with method, path, headers, version, body, and parameters.
type Request struct {
	// Method is the HTTP method (e.g., GET, POST, PUT).
	Method string

	// Path is the request path or endpoint.
	Path string

	// Headers stores HTTP headers for the request.
	Headers Headers

	// Version represents the HTTP version (e.g., HTTP/1.1).
	Version string

	// Body holds the request body, wrapped in a buffered reader.
	Body *buffer.BuffReader

	// Params contains request parameters (e.g., query or path parameters).
	Params Params
}

// NewRequest creates and returns a new empty Request instance.
func NewRequest() *Request {
	return &Request{
		Headers: make(Headers),
		Params:  make(Params),
	}
}

// SetBody initializes the request body by reading the "Content-Length" header.
//
// If the header is missing or contains an invalid value, an error is logged.
// The body is wrapped in a BuffReader for efficient reading.
func (r *Request) SetBody(body io.Reader) {
	if body == nil {
		return
	}

	// Retrieve the Content-Length header value.
	lengthStr, err := r.Headers.Get("Content-Length")
	if err != nil {
		log.Error(err.Error())
		return
	}

	// Convert Content-Length to an integer.
	var length int
	length, err = strconv.Atoi(strings.TrimSpace(lengthStr))
	if err != nil {
		log.Error(err.Error())
		length = 0
	}

	// Create a new buffered reader for the body.
	br, err := buffer.NewBuffReader(body, length)
	if err != nil {
		log.Error(err.Error(), length)
		return
	}

	r.Body = br
}

// Parser parses raw HTTP headers from a byte slice and extracts the request method, path, and headers.
//
// The first line is expected to be the request line (e.g., "GET /path HTTP/1.1").
// Subsequent lines are parsed as HTTP headers.
func (r *Request) Parser(headersByte []byte) error {
	rawHeaders := string(headersByte)
	lines := strings.Split(rawHeaders, "\r\n")

	// Parse the request line (method, path, version).
	titleParts := strings.Split(lines[0], " ")
	if len(titleParts) > 0 && len(titleParts) == 3 {
		r.Method = titleParts[0]
		r.Path = strings.TrimSpace(titleParts[1])
		r.Version = titleParts[2]
	}

	// Parse HTTP headers.
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		parts := strings.SplitN(line, ":", 2)

		if len(parts) == 2 {
			key := textproto.TrimString(parts[0])
			value := textproto.TrimString(parts[1])
			valueTrim, found := strings.CutPrefix(value, " ")
			if !found {
				r.Headers.Add(key, value)
				continue
			}

			r.Headers.Add(key, valueTrim)
		} else {
			return ErrInvalidHeader
		}
	}

	return nil
}

// Add inserts a new key-value pair into Params.
//
// If the key already exists, an error is returned.
func (p Params) Add(key, value string) error {
	_, err := p.Get(key)
	if err == nil {
		return ErrAlreadyExists
	}

	p[key] = value
	return nil
}

// Get retrieves a value from Params by key.
//
// If the key does not exist, an error is returned.
func (p Params) Get(key string) (string, error) {
	value := p[key]
	if value == "" {
		return "", ErrNotExist
	}

	return value, nil
}

// Set updates or adds a key-value pair in Params.
func (p Params) Set(key, value string) {
	p[key] = value
}

// Del removes a key from Params.
//
// If the key does not exist, an error is returned.
func (p Params) Del(key string) error {
	_, err := p.Get(key)
	if err != nil {
		return err
	}

	delete(p, key)
	return nil
}

// Add inserts a new header into Headers.
//
// If the key already exists, an error is returned.
func (h Headers) Add(key, value string) error {
	key = strings.ToLower(key)
	_, err := h.Get(key)
	if err == nil {
		return ErrAlreadyExists
	}

	h[key] = value
	return nil
}

// Get retrieves a header value by key.
//
// If the key does not exist, an error is returned.
func (h Headers) Get(key string) (string, error) {
	key = strings.ToLower(key)
	value := h[key]
	if value == "" {
		return "", ErrNotExist
	}

	return value, nil
}

// Set updates or adds a header in Headers.
func (h Headers) Set(key, value string) {
	key = strings.ToLower(key)
	h[key] = value
}

// Del removes a header from Headers.
//
// If the header does not exist, an error is returned.
func (h Headers) Del(key string) error {
	key = strings.ToLower(key)
	_, err := h.Get(key)
	if err != nil {
		return err
	}

	delete(h, key)
	return nil
}
