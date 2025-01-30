package buffer

import (
	"errors"
	"io"
)

type BuffReader struct {
	Reader    io.Reader
	chunkSize int
	len       int
	maxSize   int
}

var (
	ErrNotHaveLen = errors.New("invalid length")
	// ErrReaderIsNil = errors.New("reader is nil.")
	ErrBodyMaxSize = errors.New("body exceeds max allowed size")
	ErrInvalidRead = errors.New("read did not return full data")
)

func NewBuffReader(reader io.Reader, len int) (*BuffReader, error) {
	if len <= 0 {
		return nil, ErrNotHaveLen
	}

	return &BuffReader{
		Reader:    reader,
		len:       len,
		maxSize:   10 << 20,
		chunkSize: 4096,
	}, nil
}

func (br *BuffReader) SetMaxSize(size int) {
	br.maxSize = size
}

func (br *BuffReader) Read() ([]byte, error) {
	if br == nil {
		return nil, nil
	}
	
	if br.len > br.maxSize {
		return nil, ErrBodyMaxSize
	}

	buf := make([]byte, br.len)
	read := 0

	for read < br.len {
		chunk := br.chunkSize
		if remaining := br.len - read; chunk > remaining {
			chunk = remaining
		}

		n, err := br.Reader.Read(buf[read : read+chunk])
		read += n

		if err != nil {
			if err == io.EOF && read < br.len {
				return nil, io.ErrUnexpectedEOF
			}

			return nil, err
		}

		// if n != chunk && read < br.len {
		// 	return nil, ErrInvalidRead
		// }
	}

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
