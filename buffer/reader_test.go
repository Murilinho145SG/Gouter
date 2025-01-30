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

	reader, err := NewBuffReader(bytes.NewReader(b), len(b))
	if err != nil {
		t.Error(err)
	}
	t.Run("BufReader in one exec", func(t *testing.T) {
		r, err := reader.Read()
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, b, r)
	})

	reader, err = NewBuffReader(bytes.NewReader(b), len(b))
	if err != nil {
		t.Error(err)
	}

	t.Run("BufReader in chunks", func(t *testing.T) {
		r, err := reader.Read()
		if err != nil {
			t.Error(err)
		}
		assert.Equal(t, b, r)
	})
}
