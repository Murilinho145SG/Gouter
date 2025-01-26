package httpio

import (
	"encoding/json"
	"net"

	"github.com/Murilinho145SG/gouter/gouter/log"
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
	if w.response.Code != 0 {
		log.WarnSkip(1, "This is superfluous. You cannot declare WriteHeader more than once")
		return
	}

	w.response.Code = statusCode
}

func (w *Writer) Write(value []byte) {
	w.response.Body = append(w.response.Body, value...)
}

func (w *Writer) WriteJson(value any, indent bool) error {
	var bytes []byte
	if !indent {
		b, err := json.Marshal(value)
		if err != nil {
			return err
		}

		bytes = append(bytes, b...)
	} else {
		b, err := json.MarshalIndent(value, "", " ")
		if err != nil {
			return err
		}

		bytes = append(bytes, b...)
	}

	w.Write(bytes)

	return nil
}
