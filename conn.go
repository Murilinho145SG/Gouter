package gouter

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/Murilinho145SG/gouter/httpio"
	"github.com/Murilinho145SG/gouter/log"
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

func RunTLS(addrs string, router *Router, certStr, key string, server ...Server) error {
	cert, err := tls.LoadX509KeyPair(certStr, key)
	if err != nil {
		return err
	}

	config := &tls.Config{
		Certificates: []tls.Certificate{cert},
	}

	listener, err := net.Listen("tcp", addrs)
	if err != nil {
		return err
	}

	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		tlsConn := tls.Server(conn, config)

		go handleConn(tlsConn, router, server...)
	}
}

func parseRoute(routes HandlersList, req *httpio.Request) string {
	var originalPath string

	for k := range routes {
		partsReq := strings.Split(strings.Trim(req.Path, "/"), "/")
		parts := strings.Split(strings.Trim(k, "/"), "/")

		if len(parts) != len(partsReq) {
			continue
		}

		var matched = true
		var currentPath string

		for i := 0; i < len(parts); i++ {
			part := parts[i]
			partReq := partsReq[i]

			if strings.HasPrefix(part, ":") {
				paramName := strings.TrimPrefix(part, ":")
				req.Params.Add(paramName, partReq)
				currentPath += "/" + part
			} else if part == partReq {
				if part != "" {
					currentPath += "/" + part
				}
			} else {
				matched = false
				break
			}
		}

		if matched {
			originalPath = currentPath
			break
		}
	}

	return originalPath
}

var DefaultBuffer = 8192

func handleConn(conn net.Conn, router *Router, server ...Server) {
	defer conn.Close()
	req := parseConn(conn, server...)
	if router.DebugMode {
		log.Debug(conn.RemoteAddr().String(), "is connecting at", req.Path)
	}
	response := httpio.NewResponse(conn)
	writer := httpio.NewWriter(&response)
	originalRoute := parseRoute(router.Routes, req)
	var handler Handler
	if originalRoute != "" {
		handler = router.Routes[originalRoute]
	} else {
		handler = router.Routes[req.Path]
	}

	if handler != nil {
		handler(writer, req)
	} else {
		conn.Write([]byte("HTTP/1.1 404\r\n\r\n"))
		return
	}

	err := response.Write()
	if err != nil {
		log.Error(err)
		return
	}
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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		for {
			deadline, _ := ctx.Deadline()
			conn.SetReadDeadline(deadline)
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

			select {
			case <-ctx.Done():
				return nil
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
