/*
Package gouter implements a lightweight HTTP router with built-in documentation generation.

Features:
- Basic path routing with parameters
- Static file serving
- Automatic API documentation
- HTTPS support
- Chunked transfer encoding handling
- Connection management
*/
package gouter

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"mime"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Murilinho145SG/gouter/log"
)

const (
    // Maximum allowed size for HTTP headers (1MB)
    defaultMaxHeaderBytes = 1 << 20
)

// server represents the HTTP server instance
type server struct {
    router *Router // Main request router
}

// RunTLS starts an HTTPS server on the specified address
// Args:
// - addrs: Server address (e.g., ":443")
// - r: Router instance
// - certStr: Path to SSL certificate file
// - key: Path to private key file
// Returns error if server fails to start
func RunTLS(addrs string, r *Router, certStr, key string) error {
    // Load TLS certificate
    cert, err := tls.LoadX509KeyPair(certStr, key)
    if err != nil {
        return err
    }

    // Configure TLS parameters
    config := &tls.Config{
        Certificates:             []tls.Certificate{cert},
        MinVersion:               tls.VersionTLS12,
        PreferServerCipherSuites: true,
        CurvePreferences:         []tls.CurveID{tls.CurveP256, tls.X25519},
    }

    // Create TCP listener
    l, err := net.Listen("tcp", addrs)
    if err != nil {
        return err
    }

    // Main connection acceptance loop
    for {
        conn, err := l.Accept()
        if err != nil {
            log.Error(err)
            continue
        }

        // Wrap connection in TLS
        tlsConn := tls.Server(conn, config)

        // Set handshake timeout
        handshakeDeadline := time.Now().Add(5 * time.Second)
        tlsConn.SetDeadline(handshakeDeadline)
        if err := tlsConn.Handshake(); err != nil {
            tlsConn.Close()
            continue
        }

        // Remove deadline after handshake
        tlsConn.SetDeadline(time.Time{})

        // Handle connection in new goroutine
        go handleConn(tlsConn, r)
    }
}

// Run starts an HTTP server on the specified address
// Args:
// - addrs: Server address (e.g., ":8080")
// - r: Router instance
// Returns error if server fails to start
func Run(addrs string, r *Router) error {
    l, err := net.Listen("tcp", addrs)
    if err != nil {
        return err
    }

    // Start documentation server
    go startDoc(r)

    // Main connection acceptance loop
    for {
        conn, err := l.Accept()
        if err != nil {
            log.Error(err)
        }

        // Handle connection in new goroutine
        go handleConn(conn, r)
    }
}

// handleConn processes an incoming HTTP connection
// Args:
// - c: Network connection
// - r: Router instance
func handleConn(c net.Conn, r *Router) {
    defer c.Close()

    // Parse HTTP request
    req, err := parserConn(c)
    if err != nil {
        log.Error(err)
        return
    }

    // Create response writer
    w := newWriter(c)

    // Find matching route handler
    handler := r.parseRoute(req)
    if handler != nil {
        handler(req, w)
    } else {
        w.code = http.StatusNotFound
    }

    // Send response if headers haven't been sent
    if !w.headersSent {
        err = w.write()
        if err != nil {
            log.Error(err)
        }
    }
}

// parserConn parses an HTTP request from a network connection
// Args:
// - c: Network connection
// Returns parsed Request or error
func parserConn(c net.Conn) (*Request, error) {
    var (
        buffer     bytes.Buffer
        headersLen int
    )

    // Read headers until we find the empty line separator
    for {
        temp := make([]byte, 4096)
        n, err := c.Read(temp)
        if err != nil {
            if errors.Is(err, io.EOF) {
                continue
            }
            return nil, err
        }

        buffer.Write(temp[:n])
        headersLen = buffer.Len()

        // Check for header termination sequence
        if bytes.Contains(buffer.Bytes(), []byte("\r\n\r\n")) {
            break
        }

        // Prevent header overflow
        if headersLen >= defaultMaxHeaderBytes {
            return nil, errors.New("headers exceed maximum size")
        }
    }

    data := buffer.Bytes()
    idx := bytes.Index(data, []byte("\r\n\r\n"))
    if idx == -1 {
        return nil, errors.New("malformed headers")
    }

    // Split headers and body
    headers := data[:idx]
    bodyStart := idx + len("\r\n\r\n")
    initialBody := data[bodyStart:]

    req := newRequest()
    if err := req.parser(headers); err != nil {
        return nil, err
    }

    // Check for chunked transfer encoding
    var isChunked bool
    if te := req.Headers.Get("transfer-encoding"); te != "" {
        isChunked = (te == "chunked")
    }

    // Create appropriate body reader
    var bodyReader io.Reader
    if isChunked {
        bodyReader = newChunkedReader(io.MultiReader(bytes.NewReader(initialBody), c))
    } else {
        // Handle content-length based body
        contentLength, _ := strconv.Atoi(req.Headers.Get("content-length"))
        if contentLength > 0 {
            remaining := int64(contentLength) - int64(len(initialBody))
            bodyReader = io.MultiReader(
                bytes.NewReader(initialBody),
                io.LimitReader(c, remaining),
            )
        } else {
            bodyReader = bytes.NewReader(initialBody)
        }
    }

    req.Body = bodyReader
    req.RemoteAddrs = c.RemoteAddr().String()

    return req, nil
}

// chunkedReader implements chunked transfer encoding decoding
type chunkedReader struct {
    r    io.Reader
    done bool
}

// newChunkedReader creates a new chunked encoding reader
func newChunkedReader(r io.Reader) io.Reader {
    return &chunkedReader{r: bufio.NewReader(r)}
}

// Read implements the io.Reader interface for chunked encoding
func (cr *chunkedReader) Read(p []byte) (n int, err error) {
    if cr.done {
        return 0, io.EOF
    }

    // Read chunk size line
    line, err := cr.readLine()
    if err != nil {
        return 0, err
    }

    // Parse chunk size
    chunkSizeHex := strings.TrimSpace(strings.Split(string(line), ";")[0])
    chunkSize, err := strconv.ParseInt(chunkSizeHex, 16, 64)
    if err != nil {
        return 0, fmt.Errorf("invalid chunk size: %v", err)
    }

    // Handle end of chunked body
    if chunkSize == 0 {
        cr.done = true
        // Read and ignore trailing headers
        for {
            line, err := cr.readLine()
            if err != nil || len(line) == 0 {
                break
            }
        }
        return 0, io.EOF
    }

    // Read chunk data
    data := make([]byte, chunkSize)
    _, err = io.ReadFull(cr.r, data)
    if err != nil {
        return 0, err
    }

    // Read chunk terminator
    if _, err := cr.readLine(); err != nil {
        return 0, err
    }

    // Copy data to output buffer
    n = copy(p, data)
    return n, nil
}

// readLine reads a CRLF-terminated line
func (cr *chunkedReader) readLine() ([]byte, error) {
    var line []byte
    for {
        b := make([]byte, 1)
        _, err := cr.r.Read(b)
        if err != nil {
            return nil, err
        }
        line = append(line, b[0])
        if len(line) >= 2 && bytes.Equal(line[len(line)-2:], []byte("\r\n")) {
            break
        }
    }
    return line[:len(line)-2], nil
}

// Headers represents HTTP headers (case-insensitive)
type Headers map[string]string

// Add adds a header value
func (h Headers) Add(key, value string) {
    h[strings.ToLower(key)] = value
}

// Get retrieves a header value
func (h Headers) Get(key string) string {
    return h[strings.ToLower(key)]
}

// Params represents route path parameters
type Params map[string]string

func (h Params) add(key, value string) {
    h[key] = value
}

// Get retrieves a path parameter value
func (h Params) Get(key string) string {
    return h[key]
}

// Request represents an HTTP request
type Request struct {
    Method      string
    Path        string  // Request path
    Headers     Headers // HTTP headers
    Version     string  // HTTP version
    Body        io.Reader
    Params      Params  // Path parameters
    RemoteAddrs string  // Client address
}

// newRequest creates a new Request instance
func newRequest() *Request {
    return &Request{
        Headers: make(Headers),
        Params:  make(Params),
    }
}

// parser parses HTTP request headers
func (r *Request) parser(headersByte []byte) error {
    lines := bytes.Split(headersByte, []byte("\r\n"))

    // Parse request line
    titleParts := bytes.Split(lines[0], []byte(" "))
    if len(titleParts) != 3 {
        return errors.New("invalid headers")
    }

    r.Method = string(titleParts[0])
    r.Path = strings.TrimSpace(string(titleParts[1]))
    r.Version = string(titleParts[2])

    // Parse headers
    for i := 1; i < len(lines); i++ {
        line := lines[i]
        parts := bytes.SplitN(line, []byte(":"), 2)
        if len(parts) != 2 {
            return errors.New("invalid header format")
        }

        key := textproto.TrimBytes(parts[0])
        value := textproto.TrimBytes(parts[1])

        // Normalize header key and value
        normalizedKey := strings.ToLower(string(key))
        normalizedValue := strings.TrimPrefix(string(value), " ")

        r.Headers.Add(normalizedKey, normalizedValue)
    }

    return nil
}

// Writer handles HTTP response generation
type Writer struct {
    code        uint      // HTTP status code
    body        []byte    // Response body
    Headers     Headers   // Response headers
    c           net.Conn  // Network connection
    headersSent bool      // Flag if headers were sent
    io.Writer
}

// newWriter creates a new response writer
func newWriter(c net.Conn) *Writer {
    return &Writer{
        c:       c,
        Headers: make(Headers),
    }
}

// WriteHeader sets the HTTP status code
func (w *Writer) WriteHeader(statusCode uint) {
    if w.code != 0 {
        log.WarnE(2, "WriteHeader called multiple times")
        return
    }
    w.code = statusCode
}

// Write implements io.Writer interface
func (w *Writer) Write(p []byte) (n int, err error) {
    if w.headersSent {
        return w.c.Write(p)
    }
    w.body = append(w.body, p...)
    return len(p), nil
}

// WriteJson serializes data to JSON and sets appropriate headers
func (w *Writer) WriteJson(body any) error {
    b, err := json.Marshal(body)
    if err != nil {
        return err
    }
    w.Headers.Add("content-type", "application/json")
    _, err = w.Write(b)
    return err
}

// write sends the full HTTP response
func (w *Writer) write() error {
    if w.headersSent {
        return nil
    }

    // Build status line
    statusLine := "HTTP/1.1 200 OK\r\n"
    if w.code != 0 {
        statusText := http.StatusText(int(w.code))
        statusLine = fmt.Sprintf("HTTP/1.1 %d %s\r\n", w.code, statusText)
    }

    // Build headers
    var headersBuilder strings.Builder
    if len(w.body) > 0 && w.Headers.Get("content-length") == "" {
        w.Headers.Add("content-length", strconv.Itoa(len(w.body)))
    }

    for k, v := range w.Headers {
        headersBuilder.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
    }

    fullHeader := statusLine + headersBuilder.String() + "\r\n"

    // Write headers and body
    _, err := w.c.Write(append([]byte(fullHeader), w.body...))
    if err != nil {
        return err
    }

    w.headersSent = true
    return nil
}

// WriteHeaders sends headers without body
func (w *Writer) WriteHeaders() error {
    if w.headersSent {
        return nil
    }

    statusLine := "HTTP/1.1 200 OK\r\n"
    if w.code != 0 {
        statusText := http.StatusText(int(w.code))
        statusLine = fmt.Sprintf("HTTP/1.1 %d %s\r\n", w.code, statusText)
    }

    var headersBuilder strings.Builder
    for k, v := range w.Headers {
        headersBuilder.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
    }

    fullHeader := statusLine + headersBuilder.String() + "\r\n"

    _, err := w.c.Write([]byte(fullHeader))
    if err != nil {
        return err
    }

    w.headersSent = true
    return nil
}

// ReceiveFile reads file contents from request body
func ReceiveFile(r *Request, path string) (*os.File, error) {
    b, err := io.ReadAll(r.Body)
    if err != nil {
        return nil, err
    }

    f, err := os.Create(path)
    if err != nil {
        return nil, err
    }

    if _, err := f.Write(b); err != nil {
        f.Close()
        return nil, err
    }

    return f, nil
}

// ListenFiles generates directory listing HTML
func ListenFiles(w *Writer, r *Request, path string) error {
    entries, err := os.ReadDir(path)
    if err != nil {
        return err
    }

    // HTML template for directory listing
    tmpl := template.Must(template.New("files").Parse(`
   	<html>
	<head>
		<title>File List</title>
	</head>
	<body>
		<h1>Files in {{.Directory}}</h1>
		<ul>
			<li><a href="../">../</a></li>
			{{range .Files}}
			<li><a href="{{$.BasePath}}/{{.Name}}{{if .IsDir}}/{{end}}">{{.Name}}{{if .IsDir}}/{{end}}</a></li>
			{{end}}
		</ul>
	</body>
	</html>
    `))

    data := struct {
        Directory string
        Files     []os.DirEntry
        BasePath  string
    }{
        Directory: path,
        Files:     entries,
        BasePath:  strings.TrimSuffix(r.Path, "/"),
    }

    return tmpl.Execute(w, data)
}

// Error sends an error response
func Error(w *Writer, err error, code uint) {
    w.WriteHeader(code)
    w.Write([]byte(err.Error()))
}

// ServerStatic configures static file serving
func ServerStatic(router *Router, basePath, fsRoot string) {
    basePath = "/" + strings.Trim(basePath, "/")
    fsRoot = filepath.Clean(fsRoot)

    // Directory listing handler
    router.Route(basePath, func(r *Request, w *Writer) {
        w.Headers.Add("Access-Control-Allow-Methods", "GET, OPTIONS")

        switch r.Method {
        case "OPTIONS":
            w.WriteHeader(200)
        case "GET":
            w.Headers.Add("Content-Type", "text/html; charset=utf-8")
            if err := ListenFiles(w, r, fsRoot); err != nil {
                Error(w, errors.New("directory listing failed"), 500)
            }
        default:
            w.WriteHeader(http.StatusMethodNotAllowed)
        }
    })

    // File serving handler
    router.Route(basePath+"/*", func(r *Request, w *Writer) {
        // Path validation and normalization
        urlPath := strings.TrimPrefix(r.Path, basePath)
        decodedPath, err := url.PathUnescape(urlPath)
        if err != nil {
            w.WriteHeader(400)
            return
        }

        filePath := filepath.Join(fsRoot, decodedPath)
        cleanPath := filepath.Clean(filePath)

        // Security check
        if !strings.HasPrefix(cleanPath, fsRoot) {
            w.WriteHeader(403)
            return
        }

        // File handling
        info, err := os.Stat(cleanPath)
        if err != nil {
            w.WriteHeader(404)
            return
        }

        if info.IsDir() {
            ListenFiles(w, r, cleanPath)
            return
        }

        // Open and serve file
        file, err := os.Open(cleanPath)
        if err != nil {
            w.WriteHeader(404)
            return
        }
        defer file.Close()

        stat, _ := file.Stat()
        w.Headers.Add("Content-Length", strconv.FormatInt(stat.Size(), 10))

        // Set MIME type
        if mimeType := mime.TypeByExtension(filepath.Ext(cleanPath)); mimeType != "" {
            w.Headers.Add("Content-Type", mimeType)
        } else {
            w.Headers.Add("Content-Type", "application/octet-stream")
        }

        w.WriteHeader(200)
        if err := w.WriteHeaders(); err != nil {
            log.Error(err)
            return
        }

        // Stream file contents
        _, err = io.Copy(w.c, file)
        if err != nil && !isClosedConnectionError(err) {
            log.Error(fmt.Errorf("error copying file: %w", err))
        }
    })
}

// isClosedConnectionError checks for common connection closure errors
func isClosedConnectionError(err error) bool {
    return strings.Contains(err.Error(), "closed") ||
        strings.Contains(err.Error(), "broken pipe") ||
        strings.Contains(err.Error(), "reset by peer")
}

// startDoc starts documentation server on port 7665
func startDoc(r *Router) {
    listener, err := net.Listen("tcp", "localhost:7665")
    if err != nil {
        log.Error(err)
        return
    }

    for {
        conn, err := listener.Accept()
        if err != nil {
            log.Error(err)
            return
        }

        go handleDocRequest(conn, r)
    }
}

// handleDocRequest serves documentation HTML
func handleDocRequest(c net.Conn, r *Router) {
    defer c.Close()

    _, err := parserConn(c)
    if err != nil {
    	log.Error(err)
    	return
    }
    w := newWriter(c)

    // Documentation template with JSON formatting helpers
    tmpl := template.Must(template.New("docs").Funcs(template.FuncMap{
        "json": func(v interface{}) string {
            b, _ := json.MarshalIndent(v, "", "  ")
            return string(b)
        },
        "lower": strings.ToLower,
    }).Parse(docsTemplate))

    data := struct {
        Title  string
        Routes []*RouteInfo
    }{
        Title:  "Gouter Documentation",
        Routes: r.docs,
    }

    w.Headers.Add("Content-Type", "text/html; charset=utf-8")
    w.WriteHeader(200)

    if err := tmpl.Execute(w, data); err != nil {
        log.Error(err)
        return
    }

    w.write()
}

// HTML template constant omitted for brevity
const docsTemplate = `<!DOCTYPE html>
<html lang="en">

<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <!-- Inter font from Google Fonts -->
    <link href="https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;700&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-dark: #1e1e1e;
            --bg-panel: #111111;
            --bg-code: #222222;
            --text-primary: #ffffff;
            --text-secondary: #a0a0a0;
            --accent: #27b1b1;
            --border: #333333;
            --success: #4CAF50;
            --method-get: #2196F3;
            --method-post: #4CAF50;
            --method-put: #FF9800;
            --method-delete: #F44336;
            --method-patch: #9C27B0;

            /* JSON Syntax Highlighting */
            --json-key: #9cdcfe;
            --json-string: #ce9178;
            --json-boolean: #569cd6;
            --json-number: #b5cea8;
            --json-brace: #d4d4d4;
            --line-number: #858585;
        }

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: 'Inter', sans-serif;
            background-color: var(--bg-dark);
            color: var(--text-primary);
            line-height: 1.6;
        }

        .container {
            display: flex;
            min-height: 100vh;
        }

        .sidebar {
            width: 280px;
            background-color: var(--bg-panel);
            padding: 20px;
            overflow-y: auto;
            position: fixed;
            height: 100vh;
            z-index: 10;
            box-shadow: 0 0 20px rgba(0, 0, 0, 0.3);
            transition: transform 0.3s ease;
        }

        .main-content {
            flex: 1;
            padding: 40px;
            margin-left: 280px;
            max-width: 1200px;
        }

        .main-content h1 {
            margin-bottom: 30px;
            font-size: 32px;
            font-weight: 700;
            color: var(--text-primary);
            border-bottom: 1px solid var(--border);
            padding-bottom: 15px;
        }

        .sidebar h2 {
            margin-bottom: 20px;
            font-size: 22px;
            font-weight: 600;
            color: var(--text-primary);
        }

        .sidebar h3 {
            margin-top: 25px;
            margin-bottom: 15px;
            font-size: 14px;
            color: var(--text-secondary);
            text-transform: uppercase;
            letter-spacing: 1.5px;
            font-weight: 600;
        }

        .sidebar ul {
            list-style: none;
        }

        .sidebar li {
            margin-bottom: 10px;
        }

        .sidebar a {
            color: var(--text-primary);
            text-decoration: none;
            display: block;
            padding: 8px 12px;
            border-radius: 6px;
            transition: all 0.2s ease;
            font-size: 14px;
        }

        .sidebar a:hover {
            background-color: rgba(255, 255, 255, 0.1);
            transform: translateX(2px);
        }

        .sidebar a.active {
            background-color: var(--accent);
            color: white;
            box-shadow: 0 2px 8px rgba(153, 102, 204, 0.3);
        }

        .endpoint-card {
            background-color: var(--bg-panel);
            border-radius: 10px;
            margin-bottom: 40px;
            overflow: hidden;
            box-shadow: 0 4px 20px rgba(0, 0, 0, 0.25);
            border: 1px solid var(--border);
            transition: transform 0.2s ease, box-shadow 0.2s ease;
        }

        .endpoint-card:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 25px rgba(0, 0, 0, 0.3);
        }

        .endpoint-header {
            padding: 20px;
            border-bottom: 1px solid var(--border);
            display: flex;
            align-items: center;
            justify-content: space-between;
            background-color: rgba(0, 0, 0, 0.2);
        }

        .endpoint-method {
            font-weight: bold;
            padding: 6px 12px;
            border-radius: 6px;
            font-size: 14px;
            text-transform: uppercase;
            box-shadow: 0 2px 5px rgba(0, 0, 0, 0.2);
        }

        .method-get {
            background-color: var(--method-get);
            color: white;
        }

        .method-post {
            background-color: var(--method-post);
            color: white;
        }

        .method-put {
            background-color: var(--method-put);
            color: white;
        }

        .method-delete {
            background-color: var(--method-delete);
            color: white;
        }

        .method-patch {
            background-color: var(--method-patch);
            color: white;
        }

        .endpoint-path {
            font-family: 'Consolas', 'Monaco', monospace;
            font-size: 16px;
            margin-left: 15px;
            flex: 1;
            padding: 5px 10px;
            background-color: rgba(0, 0, 0, 0.2);
            border-radius: 4px;
            overflow-x: auto;
            white-space: nowrap;
            position: relative;
            display: flex;
            align-items: center;
        }

        .endpoint-body {
            padding: 25px;
        }

        .endpoint-description {
            margin-bottom: 25px;
            line-height: 1.7;
            color: var(--text-secondary);
            font-size: 15px;
        }

        .section-title {
            font-size: 18px;
            margin: 25px 0 15px 0;
            color: var(--text-primary);
            font-weight: 600;
        }

        .params-table {
            width: 100%;
            border-collapse: collapse;
            margin: 15px 0 25px 0;
            border-radius: 8px;
            overflow: hidden;
            box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
        }

        .params-table th {
            background-color: var(--bg-code);
            text-align: left;
            padding: 12px 15px;
            font-weight: 600;
            color: var(--text-secondary);
            font-size: 14px;
            text-transform: uppercase;
            letter-spacing: 0.5px;
        }

        .params-table td {
            padding: 12px 15px;
            border-top: 1px solid var(--border);
            font-size: 14px;
        }

        .params-table tr:hover td {
            background-color: rgba(255, 255, 255, 0.03);
        }

        .param-name {
            font-family: 'Consolas', 'Monaco', monospace;
            color: var(--json-key);
            font-weight: 500;
        }

        .param-type {
            color: var(--json-boolean);
            font-size: 13px;
            font-weight: 500;
            background-color: rgba(86, 156, 214, 0.1);
            padding: 2px 6px;
            border-radius: 4px;
        }

        .tabs {
            display: flex;
            background-color: var(--bg-code);
            border-top-left-radius: 8px;
            border-top-right-radius: 8px;
            overflow: hidden;
            border: 1px solid var(--border);
            border-bottom: none;
        }

        .tab {
            padding: 12px 20px;
            cursor: pointer;
            transition: all 0.2s ease;
            font-size: 14px;
            font-weight: 500;
            position: relative;
        }

        .tab:hover {
            background-color: rgba(255, 255, 255, 0.05);
        }

        .tab.active {
            font-weight: 600;
            color: var(--accent);
            background-color: rgba(153, 102, 204, 0.1);
        }

        .tab.active::after {
            content: '';
            position: absolute;
            bottom: 0;
            left: 0;
            right: 0;
            height: 3px;
            background-color: var(--accent);
        }

        .code-block {
            background-color: var(--bg-code);
            border-bottom-left-radius: 8px;
            border-bottom-right-radius: 8px;
            padding: 20px;
            position: relative;
            overflow-x: auto;
            border: 1px solid var(--border);
            border-top: none;
        }

        .tab-content {
            display: none;
        }

        .tab-content.active {
            display: block;
        }

        .code-block pre {
            font-family: 'Consolas', 'Monaco', monospace;
            margin: 0;
            tab-size: 4;
            font-size: 14px;
            line-height: 1.6;
        }

        .copy-btn {
            background-color: rgba(255, 255, 255, 0.1);
            border: none;
            color: var(--text-primary);
            border-radius: 4px;
            padding: 6px 12px;
            font-size: 12px;
            cursor: pointer;
            transition: all 0.2s ease;
            display: flex;
            align-items: center;
            gap: 5px;
        }

        /* Corrigindo o botão Copy URL */
        .endpoint-header .copy-btn {
            position: relative;
            top: auto;
            right: auto;
        }

        /* Estilo específico para o botão dentro do code-block */
        .code-block .copy-btn {
            position: absolute;
            top: 10px;
            right: 10px;
        }

        .copy-btn::before {
            content: '📋';
            font-size: 14px;
        }

        .copy-btn:hover {
            background-color: rgba(255, 255, 255, 0.2);
            transform: translateY(-1px);
        }

        .line-number {
            color: var(--line-number);
            text-align: right;
            padding-right: 15px;
            user-select: none;
            opacity: 0.6;
            min-width: 30px;
            display: inline-block;
        }

        .json-key {
            color: var(--json-key);
        }

        .json-string {
            color: var(--json-string);
        }

        .json-boolean {
            color: var(--json-boolean);
        }

        .json-number {
            color: var(--json-number);
        }

        .json-brace {
            color: var(--json-brace);
        }

        /* Fixed: Anchor link positioning */
        .header-anchor {
            display: flex;
            align-items: center;
            flex: 1;
        }

        .header-anchor a {
            margin-left: 8px;
            color: var(--accent);
            font-size: 18px;
            opacity: 0;
            transition: opacity 0.2s;
            text-decoration: none;
        }

        .endpoint-path:hover a {
            opacity: 1;
        }

        .no-routes {
            text-align: center;
            padding: 80px 20px;
            color: var(--text-secondary);
            background-color: var(--bg-panel);
            border-radius: 10px;
            box-shadow: 0 4px 20px rgba(0, 0, 0, 0.25);
        }

        .no-routes h2 {
            font-size: 24px;
            margin-bottom: 15px;
            color: var(--text-primary);
        }

        .no-routes p {
            font-size: 16px;
            max-width: 500px;
            margin: 0 auto;
        }

        .search-container {
            margin-bottom: 25px;
            position: relative;
        }

        .search-input {
            width: 100%;
            padding: 12px 15px;
            background-color: var(--bg-code);
            border: 1px solid var(--border);
            border-radius: 6px;
            color: var(--text-primary);
            font-family: 'Inter', sans-serif;
            font-size: 14px;
            transition: all 0.2s ease;
            padding-left: 35px;
        }

        .search-input:focus {
            outline: none;
            border-color: var(--accent);
            box-shadow: 0 0 0 3px rgba(153, 102, 204, 0.2);
        }

        .search-container::before {
            content: '🔍';
            position: absolute;
            left: 12px;
            top: 50%;
            transform: translateY(-50%);
            font-size: 14px;
            color: var(--text-secondary);
        }

        .search-clear {
            position: absolute;
            right: 10px;
            top: 50%;
            transform: translateY(-50%);
            background: none;
            border: none;
            color: var(--text-secondary);
            cursor: pointer;
            font-size: 16px;
            opacity: 0.7;
            transition: opacity 0.2s;
        }

        .search-clear:hover {
            opacity: 1;
        }

        .mobile-menu-toggle {
            display: none;
            position: fixed;
            top: 15px;
            left: 15px;
            z-index: 20;
            background-color: var(--accent);
            color: white;
            border: none;
            border-radius: 6px;
            padding: 10px 15px;
            cursor: pointer;
            font-size: 16px;
            box-shadow: 0 2px 10px rgba(0, 0, 0, 0.2);
            transition: all 0.2s ease;
        }

        .mobile-menu-toggle:hover {
            background-color: #8a57b9;
        }

        .notification {
            position: fixed;
            bottom: 25px;
            right: 25px;
            background-color: var(--success);
            color: white;
            padding: 12px 20px;
            border-radius: 6px;
            box-shadow: 0 4px 15px rgba(0, 0, 0, 0.3);
            transform: translateY(100px);
            opacity: 0;
            transition: transform 0.3s ease, opacity 0.3s ease;
            z-index: 100;
            font-weight: 500;
            display: flex;
            align-items: center;
            gap: 8px;
        }

        .notification::before {
            content: '✓';
            font-weight: bold;
            font-size: 16px;
        }

        .notification.show {
            transform: translateY(0);
            opacity: 1;
        }

        /* Scrollbar styling */
        ::-webkit-scrollbar {
            width: 8px;
            height: 8px;
        }

        ::-webkit-scrollbar-track {
            background: var(--bg-dark);
        }

        ::-webkit-scrollbar-thumb {
            background: var(--border);
            border-radius: 4px;
        }

        ::-webkit-scrollbar-thumb:hover {
            background: #555;
        }

        /* Visual enhancements */
        .endpoint-card {
            border-left: 4px solid var(--accent);
        }

        .endpoint-method {
            transform: translateY(0);
            transition: transform 0.2s ease;
        }

        .endpoint-method:hover {
            transform: translateY(-2px);
        }

        .sidebar {
            border-right: 1px solid var(--border);
        }

        .main-content h1 {
            background: linear-gradient(90deg, var(--accent), #4dffd8);
            -webkit-background-clip: text;
            background-clip: text;
            color: transparent;
            display: inline-block;
        }

        @media (max-width: 768px) {
            .mobile-menu-toggle {
                display: block;
            }

            .sidebar {
                transform: translateX(-100%);
                width: 85%;
                max-width: 320px;
            }

            .sidebar.open {
                transform: translateX(0);
            }

            .main-content {
                margin-left: 0;
                padding: 30px 20px;
                padding-top: 70px;
            }

            .endpoint-header {
                flex-direction: column;
                align-items: flex-start;
            }

            .endpoint-path {
                margin-left: 0;
                margin-top: 12px;
                width: 100%;
            }

            .endpoint-header .copy-btn {
                margin-top: 15px;
                align-self: flex-end;
            }

            .endpoint-card {
                margin-bottom: 30px;
            }

            .tabs {
                overflow-x: auto;
            }

            .tab {
                padding: 10px 15px;
                white-space: nowrap;
            }
        }

        /* Tema claro/escuro toggle */
        .theme-toggle {
            position: fixed;
            bottom: 20px;
            left: 20px;
            background-color: var(--bg-code);
            border: 1px solid var(--border);
            color: var(--text-primary);
            border-radius: 50%;
            width: 40px;
            height: 40px;
            display: flex;
            align-items: center;
            justify-content: center;
            cursor: pointer;
            z-index: 100;
            font-size: 20px;
            box-shadow: 0 2px 10px rgba(0, 0, 0, 0.2);
            transition: all 0.2s ease;
        }

        .theme-toggle:hover {
            transform: rotate(30deg);
        }

        /* Animações */
        @keyframes fadeIn {
            from {
                opacity: 0;
                transform: translateY(10px);
            }

            to {
                opacity: 1;
                transform: translateY(0);
            }
        }

        .endpoint-card {
            animation: fadeIn 0.3s ease-out;
        }

        /* Additional animations */
        @keyframes pulse {
            0% { transform: scale(1); }
            50% { transform: scale(1.05); }
            100% { transform: scale(1); }
        }

        .endpoint-method {
            animation: pulse 2s infinite;
        }

        /* Gradient borders for cards */
        .endpoint-card {
            position: relative;
            border-left: none;
        }

        .endpoint-card::before {
            content: '';
            position: absolute;
            top: 0;
            left: 0;
            width: 4px;
            height: 100%;
            background: linear-gradient(to bottom, var(--accent), #7c4dff);
            border-top-left-radius: 10px;
            border-bottom-left-radius: 10px;
        }
    </style>
</head>

<body>
    <button class="mobile-menu-toggle" id="mobileMenuToggle">☰ Menu</button>
    <button class="theme-toggle" id="themeToggle">🌙</button>

    <div class="container">
        <div class="sidebar" id="sidebar">
            <h2>{{.Title}}</h2>

            <div class="search-container">
                <input type="text" id="searchInput" class="search-input" placeholder="Search endpoints...">
                <button id="searchClear" class="search-clear">✕</button>
            </div>

            <h3>Endpoints</h3>
            <ul id="endpointsList">
                {{ range .Routes }}
                <li data-path="{{ .Path }}" data-method="{{ .Method }}">
                    <a href="#{{ .Method | lower }}-{{ .Path }}">
                        <span class="endpoint-method method-{{ .Method | lower }}">{{ .Method }}</span>
                        {{ .Path }}
                    </a>
                </li>
                {{ end }}
            </ul>
        </div>

        <div class="main-content">
            <h1>API Documentation</h1>

            {{ if .Routes }}
            {{ range .Routes }}
            <div class="endpoint-card" id="{{ .Method | lower }}-{{ .Path }}" data-path="{{ .Path }}"
                data-method="{{ .Method }}">
                <div class="endpoint-header">
                    <div class="header-anchor">
                        <span class="endpoint-method method-{{ .Method | lower }}">{{ .Method }}</span>
                        <span class="endpoint-path">
                            {{ .Path }}
                            <a href="#{{ .Method | lower }}-{{ .Path }}">#</a>
                        </span>
                    </div>
                    <button class="copy-btn" onclick="copyToClipboard('{{ .Path }}')">Copy URL</button>
                </div>

                <div class="endpoint-body">
                    {{ if .Description }}
                    <div class="endpoint-description">
                        {{ .Description }}
                    </div>
                    {{ end }}

                    {{ if .Parameters }}
                    <h3 class="section-title">Parameters</h3>
                    <table class="params-table">
                        <thead>
                            <tr>
                                <th>Name</th>
                                <th>Type</th>
                                <th>Description</th>
                            </tr>
                        </thead>
                        <tbody>
                            {{ range .Parameters }}
                            <tr>
                                <td class="param-name">{{ .Name }}</td>
                                <td class="param-type">{{ .Type }}</td>
                                <td>{{ .Description }}</td>
                            </tr>
                            {{ end }}
                        </tbody>
                    </table>
                    {{ end }}
                </div>
            </div>
            {{ end }}
            {{ else }}
            <div class="no-routes">
                <h2>No routes defined yet</h2>
                <p>Add routes to your router to see them documented here.</p>
            </div>
            {{ end }}
        </div>
    </div>

    <div class="notification" id="notification">Copied to clipboard!</div>

    <script>
        document.addEventListener('DOMContentLoaded', () => {
            // Mobile menu
            document.getElementById('mobileMenuToggle')
                .addEventListener('click', () => document.getElementById('sidebar').classList.toggle('open'));

            // Tab groups
            const allTabGroups = document.querySelectorAll('.tabs');
            allTabGroups.forEach(tabGroup => {
                const groupName = tabGroup.dataset.tabGroup;
                const tabs = Array.from(tabGroup.querySelectorAll('.tab'));
                const contents = Array.from(document.querySelectorAll(` + "`.tab-content[data-tab-group=\"${groupName}\"]`" + `));

                tabs.forEach(tab => {
                    tab.addEventListener('click', () => {
                        // Ativa a aba clicada
                        tabs.forEach(t => t.classList.remove('active'));
                        tab.classList.add('active');

                        // Exibe o conteúdo correspondente
                        const tabId = tab.dataset.tab;
                        contents.forEach(content => {
                            content.classList.toggle('active', content.id === tabId);
                        });
                    });
                });
            });

            // Search functionality
            const searchInput = document.getElementById('searchInput');
            const searchClear = document.getElementById('searchClear');
            const endpointsList = document.getElementById('endpointsList');
            const endpointCards = document.querySelectorAll('.endpoint-card');

            searchInput.addEventListener('input', () => filter(searchInput.value));
            searchClear.addEventListener('click', () => {
                searchInput.value = '';
                filter('');
            });

            function filter(term) {
                const q = term.toLowerCase();
                endpointsList.querySelectorAll('li').forEach(li => {
                    const path = li.dataset.path.toLowerCase();
                    const m = li.dataset.method.toLowerCase();
                    li.style.display = (path.includes(q) || m.includes(q)) ? '' : 'none';
                });
                endpointCards.forEach(card => {
                    const path = card.dataset.path.toLowerCase();
                    const m = card.dataset.method.toLowerCase();
                    card.style.display = (path.includes(q) || m.includes(q)) ? '' : 'none';
                });
            }

            // Highlight active sidebar item based on hash
            function highlight() {
                document.querySelectorAll('.sidebar a').forEach(a => {
                    a.classList.toggle('active', a.getAttribute('href') === window.location.hash);
                });
            }

            window.addEventListener('hashchange', highlight);
            highlight();

            // Theme toggle functionality
            const themeToggle = document.getElementById('themeToggle');
            const root = document.documentElement;

            // Check for saved theme preference
            const savedTheme = localStorage.getItem('theme');
            if (savedTheme === 'light') {
                enableLightTheme();
            }

            themeToggle.addEventListener('click', () => {
                if (themeToggle.textContent === '🌙') {
                    enableLightTheme();
                    localStorage.setItem('theme', 'light');
                } else {
                    enableDarkTheme();
                    localStorage.setItem('theme', 'dark');
                }
            });

            function enableLightTheme() {
                root.style.setProperty('--bg-dark', '#f5f5f7');
                root.style.setProperty('--bg-panel', '#ffffff');
                root.style.setProperty('--bg-code', '#f5f5f7');
                root.style.setProperty('--text-primary', '#333333');
                root.style.setProperty('--text-secondary', '#666666');
                root.style.setProperty('--border', '#e0e0e0');
                themeToggle.textContent = '☀️';
            }

            function enableDarkTheme() {
                root.style.setProperty('--bg-dark', '#1e1e1e');
                root.style.setProperty('--bg-panel', '#111111');
                root.style.setProperty('--bg-code', '#222222');
                root.style.setProperty('--text-primary', '#ffffff');
                root.style.setProperty('--text-secondary', '#a0a0a0');
                root.style.setProperty('--border', '#333333');
                themeToggle.textContent = '🌙';
            }
        });

        function showNotification(msg) {
            const n = document.getElementById('notification');
            n.textContent = msg;
            n.classList.add('show');
            setTimeout(() => n.classList.remove('show'), 2000);
        }

        function copyToClipboard(text) {
            navigator.clipboard.writeText(text)
                .then(() => showNotification('URL copied to clipboard!'))
                .catch(() => alert('Failed to copy URL'));
        }

        function copyCode(btn) {
            const code = btn.nextElementSibling.innerText.replace(/^\s*\d+\s/gm, '');
            navigator.clipboard.writeText(code)
                .then(() => showNotification('Code copied!'))
                .catch(() => alert('Failed to copy code'));
        }
    </script>
</body>

</html>
`
