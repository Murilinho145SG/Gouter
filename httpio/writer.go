package httpio

import (
	"encoding/json"

	"github.com/Murilinho145SG/gouter/log"
)

// Writer is a utility struct that simplifies writing HTTP responses.
// It encapsulates a reference to a Response and provides methods to manipulate headers and the response body.
type Writer struct {
	response *Response // Reference to the Response being written
}

// NewWriter creates and returns a new instance of Writer, associated with the provided Response.
func NewWriter(response *Response) Writer {
	return Writer{
		response: response, // Initializes the Writer with the given Response
	}
}

// Headers returns the headers of the associated Response.
func (w *Writer) Headers() Headers {
	return w.response.Headers
}

// WriteHeader sets the HTTP status code for the response.
// If the status code has already been set, a warning is logged.
func (w *Writer) WriteHeader(statusCode uint) {
	if w.response.Code != 0 {
		log.WarnSkip(1, "This is superfluous. You cannot declare WriteHeader more than once")
		return
	}

	w.response.Code = statusCode // Sets the status code
}

// Write appends data to the response body.
// If the status code has not been set, it defaults to 200 (OK) and logs a warning.
// If the body already contains data, a warning is logged to avoid overwriting.
func (w *Writer) Write(value []byte) {
	if w.response.Code == 0 {
		w.WriteHeader(200) // Defaults to status code 200 if not set
		log.WarnSkip(1, "For good practice, declare the status code before writing the body for better readability or use WriteWR")
	}

	if len(w.response.Body) > 0 {
		log.WarnSkip(1, "The body already has information saved. Configure to concatenate the body")
		return
	}

	w.response.Body = append(w.response.Body, value...) // Appends data to the response body
}

// WriteWR is a convenience method to set the status code and write the body in one call.
func (w *Writer) WriteWR(value []byte, statusCode uint) {
	w.WriteHeader(statusCode) // Sets the status code
	w.Write(value)            // Writes the body
}

// WriteJson serializes the provided value to JSON and writes it to the response body.
// If `indent` is true, the JSON output is formatted with indentation.
// Returns an error if JSON serialization fails.
func (w *Writer) WriteJson(value any, indent bool) error {
	var bytes []byte
	if !indent {
		// Serialize to JSON without indentation
		b, err := json.Marshal(value)
		if err != nil {
			return err
		}

		bytes = append(bytes, b...)
	} else {
		// Serialize to JSON with indentation
		b, err := json.MarshalIndent(value, "", " ")
		if err != nil {
			return err
		}

		bytes = append(bytes, b...)
	}

	w.Write(bytes) // Writes the JSON data to the response body

	return nil
}
