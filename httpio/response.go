package httpio

import (
	"fmt"
	"net"
	"net/textproto"
	"strconv"

	"github.com/Murilinho145SG/gouter/log"
)

// Response represents an HTTP response, containing a status code, body, headers, and a network connection.
type Response struct {
	Code    uint      // HTTP status code (e.g., 200, 404)
	Body    []byte    // Response body
	Headers Headers   // Response headers
	conn    net.Conn  // Network connection to send the response
}

// NewResponse creates and returns a new instance of Response with initialized headers and the provided network connection.
func NewResponse(conn net.Conn) Response {
	return Response{
		Headers: make(Headers), // Initializes headers as an empty map
		conn:    conn,          // Sets the network connection
	}
}

// Write sends the HTTP response through the network connection.
// Returns an error if the writing fails.
func (res *Response) Write() error {
	var statusLine string
	if res.Code == 0 {
		// If the status code is not defined, defaults to 404 (Not Found)
		statusLine = fmt.Sprintf("HTTP/1.1 %d\r\n", 404)
		log.Warn("No response code provided")
	} else {
		// Sets the status line with the provided code
		statusLine = fmt.Sprintf("HTTP/1.1 %d\r\n", res.Code)
	}

	var headers string
	if len(res.Headers) != 0 {
		if len(res.Body) > 0 {
			// Automatically adds the Content-Length header if the body is present
			err := res.Headers.Add("Content-Length", strconv.Itoa(len(res.Body)))
			if err != nil {
				log.WarnSkip(1, "You do not need to declare the body size. The size is already declared automatically")
				res.Headers.Del("Content-Length")
				res.Headers.Add("Content-Length", strconv.Itoa(len(res.Body)))
			}
		}

		// Iterates over the headers and formats them for the response
		for k, v := range res.Headers {
			value := textproto.TrimString(v)
			headers += fmt.Sprintf("%s: %s\r\n", k, value)
		}
	} else {
		if len(res.Body) > 0 {
			// If no headers are present, automatically adds Content-Length
			res.Headers.Add("Content-Length", strconv.Itoa(len(res.Body)))
			log.WarnSkip(1, "Headers are empty")
		}
	}

	// Constructs the complete response, including the status line, headers, and body
	resStr := fmt.Sprintf("%s%s\r\n%s", statusLine, headers, string(res.Body))

	// Logs the complete response for debugging
	log.DebugSkip(1, resStr)

	// Writes the response to the network connection
	_, err := res.conn.Write([]byte(resStr))
	if err != nil {
		return err
	}

	return nil
}