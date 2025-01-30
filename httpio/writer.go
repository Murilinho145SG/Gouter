package httpio

import (
	"encoding/json"

	"github.com/Murilinho145SG/gouter/log"
)

type Writer struct {
	response *Response
}

func NewWriter(response *Response) Writer {
	return Writer{
		response: response,
	}
}

func (w *Writer) Headers() Headers {
	return w.response.Headers
}

func (w *Writer) WriteHeader(statusCode uint) {
	if w.response.Code != 0 {
		log.WarnSkip(1, "This is superfluous. You cannot declare WriteHeader more than once")
		return
	}

	w.response.Code = statusCode
}

func (w *Writer) Write(value []byte) {
	if w.response.Code == 0 {
		w.WriteHeader(200)
		log.WarnSkip(1, "For good practice, declare the status code before writing the body for better readability or use WriteWR")
	}

	if len(w.response.Body) > 0 {
		log.WarnSkip(1, "The body already has information saved. Configure to concatenate the body")
		return
	}

	w.response.Body = append(w.response.Body, value...)
}

func (w *Writer) WriteWR(value []byte, statusCode uint) {
	w.WriteHeader(statusCode)
	w.Write(value)
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
