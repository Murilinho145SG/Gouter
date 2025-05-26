package gouter

import (
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"net"
	"strings"
	"time"
)

type WebSocket struct {
	conn    net.Conn
	headers Headers
}

type WebSocketConfig struct {
	CheckOrigin func(*Request) bool
}

type WebSocketHandler func(*WebSocket, *Request)

const (
	websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	readTimeout   = 15 * time.Second
	writeTimeout  = 15 * time.Second
)

func (r *Request) Upgrade(w *Writer, cfg WebSocketConfig) (*WebSocket, error) {
	if strings.ToLower(r.Headers.Get("Upgrade")) != "websocket" || strings.ToLower(r.Headers.Get("Connection")) != "upgrade" {
		return nil, errors.New("not a websocket handshake")
	}

	if cfg.CheckOrigin != nil && !cfg.CheckOrigin(r) {
		return nil, errors.New("origin not allowed")
	}

	clientKey := r.Headers.Get("Sec-WebSocket-Key")
	if clientKey == "" {
		return nil, errors.New("missing Sec-WebSocket-Key")
	}

	acceptKey := computeAcceptKey(clientKey)
	_, err := w.c.Write([]byte(
		"HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: " + acceptKey + "\r\n\r\n",
	))
	if err != nil {
		return nil, err
	}

	return &WebSocket{
		conn:    w.c,
		headers: r.Headers,
	}, nil
}

func computeAcceptKey(clientKey string) string {
	h := sha1.New()
	h.Write([]byte(clientKey + websocketGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (ws *WebSocket) ReadMessage() ([]byte, error) {
	ws.conn.SetReadDeadline(time.Now().Add(readTimeout))
	return readFrame(ws.conn)
}

func (ws *WebSocket) WriteMessage(message []byte) error {
	ws.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
	return writeFrame(ws.conn, message)
}

func (ws *WebSocket) Close() error {
	return ws.conn.Close()
}

func WebSocketRoute(handler WebSocketHandler, cfg WebSocketConfig) Handler {
	return func(r *Request, w *Writer) {
		ws, err := r.Upgrade(w, cfg)
		if err != nil {
			Error(w, err, 400)
			return
		}
		defer ws.Close()

		handler(ws, r)
	}
}
