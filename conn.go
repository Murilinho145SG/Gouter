package gouter

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net"
	"time"

	"github.com/Murilinho145SG/gouter/httpio"
	"github.com/Murilinho145SG/gouter/log"
)

// --- Constants for Size Limits ---

const (
	// DefaultBuffer is the default buffer size used for reading connections,
	// if no custom configuration is provided.
	DefaultBuffer = 8192

	// MaxHeaderSize defines the maximum allowed size for HTTP headers.
	// In this example, it is limited to 10 KB.
	MaxHeaderSize = 10 * 1024

	// MaxBodySize defines the maximum allowed size for the HTTP request body.
	// In this example, it is limited to 1 MB.
	MaxBodySize = 1 * 1024 * 1024
)

// Server defines configuration options for the HTTP server.
type Server struct {
	// InitialReadSize specifies the initial buffer size used for reading from the connection.
	// If set to zero, DefaultBuffer will be used.
	InitialReadSize int

	// InitialReadChunk indicates whether the connection should be read in chunks (true)
	// or in a single block (false).
	InitialReadChunk bool
}

// Run starts an HTTP server on the specified address and handles incoming connections
// using the provided router.
//
// Parameters:
//   - addrs: The address (host:port) on which the server should listen.
//   - router: The router instance that manages routes and their handlers.
//   - server: (Optional) Custom server configurations.
//
// Returns:
//   - An error if creating the listener or processing connections fails.
func Run(addrs string, router *Router, server ...Server) error {
	// Start listening on the specified address.
	listener, err := net.Listen("tcp", addrs)
	if err != nil {
		return err
	}

	// Continuously accept new connections.
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		// Handle each connection in a separate goroutine.
		go handleConn(conn, router, server...)
	}
}

// RunTLS starts an HTTPS server on the specified address using the provided TLS certificate and key.
//
// Parameters:
//   - addrs: The address (host:port) where the HTTPS server should listen.
//   - router: The router instance for managing routes.
//   - certStr: The path to the TLS certificate.
//   - key: The path to the private key corresponding to the certificate.
//   - server: (Optional) Custom server configurations.
//
// Returns:
//   - An error if TLS configuration, listener creation, or connection processing fails.
func RunTLS(addrs string, router *Router, certStr, key string, server ...Server) error {
	// Load the TLS certificate and key.
	cert, err := tls.LoadX509KeyPair(certStr, key)
	if err != nil {
		return err
	}

	// Set up TLS configurations.
	config := &tls.Config{
		Certificates:             []tls.Certificate{cert},
		MinVersion:               tls.VersionTLS12,
		PreferServerCipherSuites: true,
		CurvePreferences:         []tls.CurveID{tls.CurveP256, tls.X25519},
	}

	// Create a TCP listener on the specified address.
	listener, err := net.Listen("tcp", addrs)
	if err != nil {
		return err
	}

	// Continuously accept new connections.
	for {
		conn, err := listener.Accept()
		if err != nil {
			return err
		}

		// Wrap the connection with TLS.
		tlsConn := tls.Server(conn, config)

		// Set a deadline for the TLS handshake to prevent hanging connections.
		handshakeDeadline := time.Now().Add(5 * time.Second)
		tlsConn.SetDeadline(handshakeDeadline)
		if err := tlsConn.Handshake(); err != nil {
			log.Debug("TLS handshake failed:", err)
			tlsConn.Close()
			continue
		}
		// Remove the deadline after a successful handshake.
		tlsConn.SetDeadline(time.Time{})

		// Handle the TLS connection in a separate goroutine.
		go handleConn(tlsConn, router, server...)
	}
}

// handleConn processes a connection by reading the HTTP request and directing it
// to the appropriate handler based on the registered routes.
//
// Parameters:
//   - conn: The network connection established with the client.
//   - router: The router instance managing the route handlers.
//   - server: (Optional) Custom server configurations.
func handleConn(conn net.Conn, router *Router, server ...Server) {
	// Ensure that the connection is closed once processing is complete.
	defer conn.Close()

	// Set a deadline for the connection to protect against slowloris attacks.
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	// Parse the HTTP request from the connection.
	req := parseConn(conn, server...)
	response := httpio.NewResponse(conn)

	// If the request is invalid (e.g., headers too large or timeout), respond with 413 (Payload Too Large).
	if req == nil {
		response.Code = 413
		response.Write()
		return
	}

	// Sanitize the request path before logging to prevent log injection.
	sanitizedPath := sanitize(req.Path)
	log.Debug(conn.RemoteAddr().String(), "is connecting at", sanitizedPath)

	// Obtain the handler corresponding to the request path.
	handler := router.ParseRoute(req)
	writer := httpio.NewWriter(&response)

	// If a handler is found, invoke it; otherwise, respond with 404 (Not Found).
	if handler != nil {
		handler(writer, req)
	} else {
		conn.Write([]byte("HTTP/1.1 404\r\n\r\n"))
		return
	}

	// Write the response to the client and log any errors that occur.
	if err := response.Write(); err != nil {
		log.Debug(err)
	}
}

// parseConn reads and parses an HTTP request from the connection.
//
// This function uses custom server configurations (if provided) to determine the buffer size
// and whether to use chunked reading or single-block reading.
//
// Parameters:
//   - conn: The network connection from which the request is read.
//   - server: (Optional) Custom server configurations.
//
// Returns:
//   - An instance of httpio.Request containing the request data,
//     or nil if an error occurs during reading or parsing.
func parseConn(conn net.Conn, server ...Server) *httpio.Request {
	var buffer []byte
	var serverConfig Server

	// Use custom server configuration if provided.
	if server != nil {
		serverConfig = server[0]
	}

	// Determine the buffer size based on configuration or use the default.
	if serverConfig.InitialReadSize != 0 {
		buffer = make([]byte, serverConfig.InitialReadSize)
	} else {
		buffer = make([]byte, DefaultBuffer)
	}

	// Create a new Request instance using the client's remote address.
	req := httpio.NewRequest(conn.RemoteAddr().String())

	// If chunked reading is enabled:
	if serverConfig.InitialReadChunk {
		var readBytes []byte
		// Create a context with a timeout for reading the headers.
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		for {
			// Update the connection's read deadline based on the context.
			deadline, _ := ctx.Deadline()
			conn.SetReadDeadline(deadline)

			n, err := conn.Read(buffer)
			if err != nil {
				// If EOF is encountered, break the loop.
				if err == io.EOF {
					break
				}
				return nil
			}

			if n > 0 {
				data := buffer[:n]
				readBytes = append(readBytes, data...)

				// Verify that the accumulated header size does not exceed the allowed limit.
				if len(readBytes) > MaxHeaderSize {
					log.Debug("Header size exceeds maximum allowed")
					return nil
				}

				// Check if the end of the headers (indicated by "\r\n\r\n") has been reached.
				if bytes.Contains(readBytes, []byte("\r\n\r\n")) {
					values := bytes.SplitN(readBytes, []byte("\r\n\r\n"), 2)
					headers := values[0]

					// Parse the headers using the Request's Parser method.
					if err = req.Parser(headers); err != nil {
						log.Debug("Error parsing headers:", err)
						return nil
					}

					// Set the request body using io.LimitReader to restrict its size.
					bodyBytes := values[1]
					bodyReader := io.LimitReader(
						io.MultiReader(bytes.NewBuffer(bodyBytes), conn),
						MaxBodySize,
					)
					req.SetBody(bodyReader)
					break
				}
			}

			// If the context times out, stop reading.
			select {
			case <-ctx.Done():
				log.Debug("Timeout reading request")
				return nil
			default:
				continue
			}
		}
	} else {
		// Single-block reading.
		n, err := conn.Read(buffer)
		if err != nil && err != io.EOF {
			return nil
		}

		if n > 0 {
			data := buffer[:n]

			// Check if the header size exceeds the allowed maximum.
			if len(data) > MaxHeaderSize {
				log.Debug("Header size exceeds maximum allowed")
				return nil
			}

			// Check if the headers have been fully received.
			if bytes.Contains(data, []byte("\r\n\r\n")) {
				values := bytes.SplitN(data, []byte("\r\n\r\n"), 2)
				headers := values[0]
				if err = req.Parser(headers); err != nil {
					log.Debug("Error parsing headers:", err)
					return nil
				}

				// Set the request body, limiting its size.
				bodyBytes := values[1]
				bodyReader := io.LimitReader(
					io.MultiReader(bytes.NewBuffer(bodyBytes), conn),
					MaxBodySize,
				)
				req.SetBody(bodyReader)
			}
		}
	}

	return req
}

// sanitize removes newline and carriage return characters from the input string
// to prevent log injection.
//
// Parameters:
//   - input: The string to sanitize.
//
// Returns:
//   - The sanitized string with newline and carriage return characters replaced by spaces.
func sanitize(input string) string {
	s := bytes.ReplaceAll([]byte(input), []byte("\n"), []byte(" "))
	s = bytes.ReplaceAll(s, []byte("\r"), []byte(" "))
	return string(s)
}
