package buffer

import (
	"errors"
	"io"
)

// BuffReader is a buffered reader that reads data in chunks from an io.Reader.
//
// It allows configuring the chunk size, total length, and maximum allowed size
// to optimize data processing and prevent excessive memory usage.
type BuffReader struct {
	// Reader is the underlying io.Reader from which data is read.
	Reader io.Reader

	// chunkSize defines the size of each read operation in bytes.
	chunkSize int

	// len represents the total length of data to be read.
	len int

	// maxSize defines the maximum allowed size for the buffer.
	maxSize int
}

// Predefined errors for BuffReader operations.
var (
	// ErrNotHaveLen is returned when an invalid or zero-length buffer is provided.
	ErrNotHaveLen = errors.New("invalid length")

	// ErrReaderIsNil is returned when attempting to use a nil reader.
	ErrReaderIsNil = errors.New("reader is nil")

	// ErrBodyMaxSize is returned when the body size exceeds the maximum allowed limit.
	ErrBodyMaxSize = errors.New("body exceeds max allowed size")

	// ErrInvalidRead is returned when the read operation does not return the expected amount of data.
	ErrInvalidRead = errors.New("read did not return full data")
)

// NewBuffReader creates a new BuffReader with the specified reader and length.
//
// It initializes the reader with a default maximum size of 10MB and a default chunk
// size of 4096 bytes for efficient reading. If the provided length is zero or negative,
// it returns an error.
func NewBuffReader(reader io.Reader, len int) (*BuffReader, error) {
	// Validate the provided length to ensure it is greater than zero.
	if len <= 0 {
		return nil, ErrNotHaveLen
	}

	// Create and return a new BuffReader instance with default configurations.
	return &BuffReader{
		Reader:    reader,
		len:       len,
		maxSize:   10 << 20, // 10MB max size
		chunkSize: 4096,     // Default chunk size
	}, nil
}

// SetMaxSize updates the maximum allowed size for the BuffReader.
//
// This function allows dynamic configuration of the maximum buffer size to
// accommodate different use cases and prevent excessive memory usage.
func (br *BuffReader) SetMaxSize(size int) {
	br.maxSize = size
}

// Read reads and returns the data from the BuffReader as a byte slice.
//
// It first checks if the BuffReader instance is valid before proceeding.
// If the buffer size exceeds the maximum allowed limit, an error is returned.
// The data is read in chunks to ensure efficient reading while adhering to 
// predefined constraints.
//
// If an error occurs during reading, it may return io.ErrUnexpectedEOF if 
// the end of the file is reached before the expected amount of data is read.
func (br *BuffReader) Read() ([]byte, error) {
	// Checks if the BuffReader instance is nil and returns an error if so.
	if br == nil {
		return nil, ErrReaderIsNil
	}

	// Returns an error if the buffer size exceeds the maximum allowed size.
	if br.len > br.maxSize {
		return nil, ErrBodyMaxSize
	}

	// Allocates a byte buffer with the required size to store the data.
	buf := make([]byte, br.len)
	read := 0 // Counter for bytes read.

	// Reads the data in chunks until the entire buffer is filled.
	for read < br.len {
		chunk := br.chunkSize // Defines the chunk size for reading.
		if remaining := br.len - read; chunk > remaining {
			chunk = remaining // Adjusts the chunk size to avoid exceeding the required amount.
		}

		// Reads a portion of data from the Reader into the buffer.
		n, err := br.Reader.Read(buf[read : read+chunk])
		read += n // Updates the count of bytes read.

		// Handles any errors that occur during reading.
		if err != nil {
			// Returns a specific error if an unexpected EOF is encountered.
			if err == io.EOF && read < br.len {
				return nil, io.ErrUnexpectedEOF
			}

			// Returns any other error encountered during reading.
			return nil, err
		}

		// Commented-out section that could check for invalid reads.
		// if n != chunk && read < br.len {
		// 	return nil, ErrInvalidRead
		// }
	}

	// Returns the successfully read data.
	return buf, nil
}



// func (br *BuffReader) SetMaxSize(size int) {
// 	br.maxSize = size
// }

// func (br *BuffReader) SetChunk(chunk int) {
// 	br.chunk = chunk
// }

// func (br *BuffReader) Read() ([]byte, error) {
// 	if br.Reader == nil {
// 		return nil, ErrReaderIsNil
// 	}

// 	if br.Len > br.maxSize {
// 		return nil, ErrBodyMaxSize
// 	}

// 	if br.chunk == 0 {
// 		if br.Len == 0 {
// 			return nil, ErrNotHaveLen
// 		}
// 		bytes := make([]byte, br.Len)
// 		n, err := br.Reader.Read(bytes)
// 		if err != nil {
// 			return nil, err
// 		}

// 		return bytes[:n], nil
// 	} else {
// 		if br.Len == 0 {
// 			return nil, ErrNotHaveLen
// 		}

// 		bytes := make([]byte, br.chunk)
// 		var bytes_read []byte
// 		total_read := 0
// 		for total_read < br.Len {
// 			n, err := br.Reader.Read(bytes)
// 			if err != nil {
// 				return nil, err
// 			}

// 			total_read += n
// 			bytes_read = append(bytes_read, bytes[:n]...)
// 			if total_read == br.Len {
// 				return bytes_read, nil
// 			}
// 		}
// 		return bytes_read, nil
// 	}
// }