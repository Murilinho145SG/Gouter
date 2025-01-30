package gouter

// import (
// 	"net"
// 	"os"
// 	"testing"

// 	"github.com/Murilinho145SG/gouter/httpio"
// 	"github.com/stretchr/testify/assert"
// )

// func testConn() error {
// 	conn, err := net.Dial("tcp", "127.0.0.1:0")
// 	if err != nil {
// 		return err
// 	}
// 	defer conn.Close()

// 	b, err := os.ReadFile("./tests/data/httpio/headers.txt")
// 	if err != nil {
// 		return err
// 	}

// 	_, err = conn.Write(b)
// 	if err != nil {
// 		return err
// 	}

// 	return nil
// }

// func TestRunnerConn(t *testing.T) {
// 	b, err := os.ReadFile("./tests/data/httpio/headers.txt")
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	l, err := net.Listen("tcp", "localhost:3123")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	defer l.Close()

// 	conn, err := l.Accept()
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	defer conn.Close()

// 	if err != nil {
// 		t.Error(err)
// 	}

// 	go testConn()

// 	req := httpio.NewRequest()
// 	buffer := make([]byte, 1024)
// 	n, err := conn.Read(buffer)
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	assert.Equal(t, b, buffer[:n])
// 	err = req.Parser(buffer[:n])
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	res := httpio.NewResponse(conn)
// 	res.Body = buffer[:n]
// 	res.Code = 200
// 	res.Headers = make(httpio.Headers)
// 	res.Write()
// }
