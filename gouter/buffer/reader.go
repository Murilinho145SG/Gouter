package buffer

import (
	"errors"
	"io"
)

type BuffReader struct {
	Reader io.Reader
	Chunk  int
	Len    int
}

var (
	ErrNotHaveLen  = errors.New("i can read continuous stream without length or length == 0.")
	ErrReaderIsNil = errors.New("reader is nil.")
)

func NewBuffReader(reader io.Reader, len int) BuffReader {
	return BuffReader{
		Reader: reader,
		Len:    len,
	}
}

func (br *BuffReader) SetChunk(chunk int) {
	br.Chunk = chunk
}

func (br *BuffReader) Read() ([]byte, error) {
	if br.Reader == nil {
		return nil, ErrReaderIsNil
	}

	if br.Chunk == 0 {
		if br.Len == 0 {
			return nil, ErrNotHaveLen
		}
		bytes := make([]byte, br.Len)
		n, err := br.Reader.Read(bytes)
		if err != nil {
			return nil, err
		}

		return bytes[:n], nil
	} else {
		if br.Len == 0 {
			return nil, ErrNotHaveLen
		}

		bytes := make([]byte, br.Chunk)
		var bytes_read []byte
		total_read := 0
		for {
			n, err := br.Reader.Read(bytes)
			if err != nil {
				return nil, err
			}

			total_read += n
			bytes_read = append(bytes_read, bytes[:n]...)
			if total_read == br.Len {
				return bytes_read, nil
			}
		}
	}
}
