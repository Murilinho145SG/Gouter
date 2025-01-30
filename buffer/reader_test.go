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

// func generateData(size int) []byte {
// 	data := make([]byte, size)
// 	rand.Read(data)
// 	return data
// }

// func BenchmarkBuffReader_NoChunk(b *testing.B) {
// 	data := generateData(10 << 20) // 10MB
// 	reader := bytes.NewReader(data)

// 	b.ResetTimer() // Ignora tempo de setup

// 	for i := 0; i < b.N; i++ {
// 		br := NewBuffReader(reader, len(data))
// 		br.SetChunk(0) // Sem chunk

// 		_, err := br.Read()
// 		if err != nil {
// 			b.Fatal(err)
// 		}

// 		reader.Seek(0, 0) // Volta ao início para próxima iteração
// 	}
// }

// func BenchmarkBuffReader_4KBChunk(b *testing.B) {
// 	data := generateData(10 << 20) // 10MB
// 	reader := bytes.NewReader(data)

// 	b.ResetTimer()

// 	for i := 0; i < b.N; i++ {
// 		br := NewBuffReader(reader, len(data))
// 		br.SetChunk(4 << 10) // 4KB

// 		_, err := br.Read()
// 		if err != nil {
// 			b.Fatal(err)
// 		}

// 		reader.Seek(0, 0)
// 	}
// }

// func BenchmarkBuffReader_1MBChunk(b *testing.B) {
// 	data := generateData(10 << 20) // 10MB
// 	reader := bytes.NewReader(data)

// 	b.ResetTimer()

// 	for i := 0; i < b.N; i++ {
// 		br := NewBuffReader(reader, len(data))
// 		br.SetChunk(1 << 20) // 1MB

// 		_, err := br.Read()
// 		if err != nil {
// 			b.Fatal(err)
// 		}

// 		reader.Seek(0, 0)
// 	}
// }