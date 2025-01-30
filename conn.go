package gouter

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/Murilinho145SG/gouter/httpio"
	"github.com/Murilinho145SG/gouter/log"
)

// Server defines configuration options for the HTTP server.
type Server struct {
	InitialReadSize  int  // Initial buffer size for reading from the connection
	InitialReadChunk bool // Whether to read data in chunks or as a single block
}

// Run starts an HTTP server on the specified address and handles incoming connections using the provided router.
func Run(addrs string, router *Router, server ...Server) error {
	// Start listening on the specified address
	listener, err := net.Listen("tcp", addrs)
	if err != nil {
		return err
	}

	// Accept and handle incoming connections in a loop
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		// Handle each connection in a separate goroutine
		go handleConn(conn, router, server...)
	}
}

// RunTLS starts an HTTPS server on the specified address using the provided TLS certificate and key.
func RunTLS(addrs string, router *Router, certStr, key string, server ...Server) error {
	// Load the TLS certificate and key
	cert, err := tls.LoadX509KeyPair(certStr, key)
	if err != nil {
		return err
	}

	// Configure TLS settings
	config := &tls.Config{
		Certificates:             []tls.Certificate{cert},                  // Server certificate
		MinVersion:               tls.VersionTLS12,                         // Minimum TLS version
		PreferServerCipherSuites: true,                                     // Prefer server cipher suites
		CurvePreferences:         []tls.CurveID{tls.CurveP256, tls.X25519}, // Preferred elliptic curves
	}

	// Start listening on the specified address
	listener, err := net.Listen("tcp", addrs)
	if err != nil {
		return err
	}

	// Accept and handle incoming connections in a loop
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		// Wrap the connection in a TLS server
		tlsConn := tls.Server(conn, config)

		// Handle each connection in a separate goroutine
		go handleConn(tlsConn, router, server...)
	}
}

// DefaultBuffer is the default buffer size for reading from connections.
var DefaultBuffer = 8192

// handleConn processes an incoming connection, parses the HTTP request, and routes it to the appropriate handler.
func handleConn(conn net.Conn, router *Router, server ...Server) {
	defer conn.Close() // Ensure the connection is closed when done

	// Set a deadline for the connection to prevent hanging
	conn.SetDeadline(time.Now().Add(time.Second * 10))

	// Parse the HTTP request from the connection
	req := parseConn(conn, server...)

	// Find the appropriate handler for the request path
	handler := router.ParseRoute(req)

	// Create a new HTTP response
	response := httpio.NewResponse(conn)

	// If the request is invalid, respond with a 413 (Payload Too Large) status
	if req == nil {
		response.Code = 413
		response.Write()
		return
	}

	// Create a writer for the response
	writer := httpio.NewWriter(&response)

	// Log the connection details
	log.Debug(conn.RemoteAddr().String(), "is connecting at", req.Path)

	// If a handler is found, invoke it; otherwise, respond with a 404 (Not Found) status
	if handler != nil {
		handler(writer, req)
	} else {
		conn.Write([]byte("HTTP/1.1 404\r\n\r\n"))
		return
	}

	// Write the response to the connection
	err := response.Write()
	if err != nil {
		log.Debug(err)
	}

	return
}

// parseConn reads and parses an HTTP request from the connection.
func parseConn(conn net.Conn, server ...Server) *httpio.Request {
	var buffer []byte
	var serverConfig Server

	// Use the provided server configuration if available
	if server != nil {
		serverConfig = server[0]
	}

	// Initialize the buffer with the specified size or the default size
	if serverConfig.InitialReadSize != 0 {
		buffer = make([]byte, serverConfig.InitialReadSize)
	} else {
		buffer = make([]byte, DefaultBuffer)
	}

	// Create a new HTTP request
	req := httpio.NewRequest()

	// If chunked reading is enabled, read the request in chunks
	if serverConfig.InitialReadChunk {
		var read_bytes []byte
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		for {
			deadline, _ := ctx.Deadline()
			conn.SetReadDeadline(deadline)

			// Read data from the connection
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

				// Check if the end of the headers is reached
				if bytes.Contains(read_bytes, []byte("\r\n\r\n")) {
					values := bytes.SplitN(read_bytes, []byte("\r\n\r\n"), 2)
					headers := values[0]

					// Parse the headers
					err = req.Parser(headers)
					if err != nil {
						fmt.Println(err.Error())
						return nil
					}

					// Set the request body
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
			default:
				continue
			}
		}
	} else {
		// Read the request in a single block
		n, err := conn.Read(buffer)
		if err != nil {
			if err == io.EOF {
				return req
			}

			return nil
		}

		if n > 0 {
			data := buffer[:n]

			// Check if the end of the headers is reached
			if bytes.Contains(data, []byte("\r\n\r\n")) {
				values := bytes.SplitN(data, []byte("\r\n\r\n"), 2)
				headers := values[0]

				// Parse the headers
				err = req.Parser(headers)
				if err != nil {
					fmt.Println(err.Error())
					return nil
				}

				// Set the request body
				bodyBytes := values[1]
				bodyBuffer := bytes.NewBuffer(bodyBytes)
				bodyReader := io.MultiReader(bodyBuffer, conn)
				req.SetBody(bodyReader)
			}
		}
	}

	return req
}
