package httpio

import (
	"net"
)

type Writer struct {
	stream   net.Conn
	response *Response
	Headers
}

func NewWriter(stream net.Conn, response *Response) Writer {
	return Writer{
		stream:   stream,
		response: response,
		Headers:  make(Headers),
	}
}

func (w *Writer) WriteHeader(statusCode uint) {
	w.response.Code = statusCode
}

func (w *Writer) Write(value []byte) {
	w.response.Body = append(w.response.Body, value...)
}
