package httpio

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParser(t *testing.T) {
	b, err := os.ReadFile("../tests/data/httpio/headers.txt")
	if err != nil {
		t.Error(err)
	}

	req := NewRequest("")
	t.Run("Parsing headers", func(t *testing.T) {
		err := req.Parser(b)
		if err != nil {
			t.Error(err)
		}

		lines := strings.Split(string(b), "\r\n")
		titleParts := strings.Split(lines[0], " ")
		assert.Equal(t, 3, len(titleParts))
		if len(titleParts) > 0 && len(titleParts) == 3 {
			assert.Equal(t, titleParts[0], req.Method)
			assert.Equal(t, titleParts[1], strings.TrimSpace(req.Path))
			assert.Equal(t, titleParts[2], req.Version)
		}

		assert.Greater(t, len(lines), 0)
		h := make(Headers)
		for i := 1; i < len(lines); i++ {
			line := lines[i]
			parts := strings.SplitN(line, ":", 2)
			assert.Equal(t, 2, len(parts))

			if len(parts) == 2 {
				key := parts[0]
				value := parts[1]
				valueTrim, found := strings.CutPrefix(value, " ")
				if !found {
					h.Add(key, value)
					continue
				}

				h.Add(key, valueTrim)
			} else {
				t.Error(ErrInvalidHeader)
			}
		}

		assert.Equal(t, len(h), len(req.Headers))
		for k, v := range h {
			value, err := req.Headers.Get(k)
			if err != nil {
				t.Error(err)
			}
			assert.Equal(t, v, value)
		}
	})
}
