package gouter

import (
	"bytes"
	"fmt"
	"io"
	"net"

	"github.com/Murilinho145SG/gouter/gouter/httpio"
)

type Server struct {
	InitialReadSize  int
	InitialReadChunk bool
}

func Run(addrs string, router *Router, server ...Server) error {
	listener, err := net.Listen("tcp", addrs)
	if err != nil {
		return err
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		go handleConn(conn, router, server...)
	}
}

var DefaultBuffer = 8192

func handleConn(conn net.Conn, router *Router, server ...Server) {
	defer conn.Close()
	req := parseConn(conn, server...)
	response := httpio.NewResponse()
	writer := httpio.NewWriter(conn, &response)
	handler := router.Routes[req.Path]
	if handler != nil {
		handler(writer, req)
	} else {
		conn.Write([]byte("HTTP/1.1 404\r\n\r\n"))
		return
	}

	statusLine := fmt.Sprintf("HTTP/1.1 %d\r\n", response.Code)
	headers := ""
	for k, v := range response.Headers {
		headers += fmt.Sprintf("%s: %s\r\n", k, v)
	}

	if len(response.Body) > 0 && response.Code == 0 {
		statusLine = fmt.Sprintf("HTTP/1.1 %d\r\n", 200)
	}

	resStr := fmt.Sprintf("%s%s\r\n%s", statusLine, headers, string(response.Body))
	conn.Write([]byte(resStr))

	return
}

func parseConn(conn net.Conn, server ...Server) *httpio.Request {
	var buffer []byte
	var serverConfig Server
	if server != nil {
		serverConfig = server[0]
	}

	if serverConfig.InitialReadSize != 0 {
		buffer = make([]byte, serverConfig.InitialReadSize)
	} else {
		buffer = make([]byte, DefaultBuffer)
	}

	req := httpio.NewRequest()
	if serverConfig.InitialReadChunk {
		var read_bytes []byte
		for {
			n, err := conn.Read(buffer)
			if err != nil {
				if err == io.EOF {
					break
				}

				return nil
			}

			if n > 0 {
				data := buffer[:n]
				read_bytes = append(read_bytes, data...)

				if bytes.Contains(read_bytes, []byte("\r\n\r\n")) {
					values := bytes.SplitN(read_bytes, []byte("\r\n\r\n"), 2)
					headers := values[0]
					err = req.Parser(headers)
					if err != nil {
						fmt.Println(err.Error())
						return nil
					}

					bodyBytes := values[1]
					bodyBuffer := bytes.NewBuffer(bodyBytes)
					bodyReader := io.MultiReader(bodyBuffer, conn)
					req.SetBody(bodyReader)
					break
				}
			}
		}
	} else {
		n, err := conn.Read(buffer)
		if err != nil {
			if err == io.EOF {
				return req
			}

			return nil
		}

		if n > 0 {
			data := buffer[:n]

			if bytes.Contains(data, []byte("\r\n\r\n")) {
				values := bytes.SplitN(data, []byte("\r\n\r\n"), 2)
				headers := values[0]
				err = req.Parser(headers)
				if err != nil {
					fmt.Println(err.Error())
					return nil
				}

				bodyBytes := values[1]
				bodyBuffer := bytes.NewBuffer(bodyBytes)
				bodyReader := io.MultiReader(bodyBuffer, conn)
				req.SetBody(bodyReader)
			}
		}
	}

	return req
}
