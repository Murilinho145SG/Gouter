package buffer

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuffReader(t *testing.T) {
	b, err := os.ReadFile("../tests/data/buffer/bufreader.json")
	if err != nil {
		t.Error(err)
	}

	reader := NewBuffReader(bytes.NewReader(b), len(b))
	t.Run("BufReader in one exec", func(t *testing.T) {
		r, err := reader.Read()
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, b, r)
	})

	reader = NewBuffReader(bytes.NewReader(b), len(b))

	t.Run("BufReader in chunks", func(t *testing.T) {
		reader.SetChunk(1)
		r, err := reader.Read()
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, b, r)
	})
}
