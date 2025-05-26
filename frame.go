package gouter

import (
	"encoding/binary"
	"errors"
	"io"
	"net"
)

func readFrame(conn net.Conn) ([]byte, error) {
	header := make([]byte, 2)
	_, err := io.ReadFull(conn, header)
	if err != nil {
		return nil, err
	}

	// fin := header[0] & 0x80
	opcode := header[0] & 0x0F
	masked := header[1] & 0x80
	payloadLen := uint64(header[1] & 0x7F)

	if payloadLen == 126 {
		lenBuf := make([]byte, 2)
		_, err := io.ReadFull(conn, lenBuf)
		if err != nil {
			return nil, err
		}
		payloadLen = uint64(binary.BigEndian.Uint16(lenBuf))
	}

	if payloadLen == 127 {
		lenBuf := make([]byte, 8)
		_, err := io.ReadFull(conn, lenBuf)
		if err != nil {
			return nil, err
		}
		payloadLen = binary.BigEndian.Uint64(lenBuf)
	}

	var maskKey []byte
	if masked == 0x80 {
		maskKey = make([]byte, 4)
		_, err := io.ReadFull(conn, maskKey)
		if err != nil {
			return nil, err
		}
	}

	payload := make([]byte, payloadLen)
	_, err = io.ReadFull(conn, payload)
	if err != nil {
		return nil, err
	}

	if masked == 0x80 {
		for i := uint64(0); i < payloadLen; i++ {
			payload[i] ^= maskKey[i%4]
		}
	}

	switch opcode {
		case 0x01:
			return payload, nil
		case 0x08:
			return nil, errors.New("close frame received")
		case 0x09:
			err = sendPong(conn, payload)
			return nil, err
		case 0x0A:
			return nil, nil
		default:
			return nil, errors.New("unsupported frame type")
	}
	
	// if fin != 0x80 || opcode != 0x01 {
	// 	return nil, errors.New("unsupported frame type")
	// }
	// return payload, nil
}

func sendPong(conn net.Conn, payload []byte) error {
	header := make([]byte, 2)
	header[0] = 0x8A
	header[1] = byte(len(payload))
	
	_, err := conn.Write(append(header, payload...))
	return err
}

func writeFrame(conn net.Conn, message []byte) error {
	header := make([]byte, 2)
	header[0] = 0x81

	payloadLen := len(message)

	if payloadLen <= 125 {
		header[1] = byte(payloadLen)
		fullMessage := append(header, message...)
		_, err := conn.Write(fullMessage)
		return err
	}

	if payloadLen <= 65535 {
		header[1] = 126
		size := make([]byte, 2)
		binary.BigEndian.PutUint16(size, uint16(payloadLen))
		header = append(header, size...)
		fullMessage := append(header, message...)
		_, err := conn.Write(fullMessage)
		return err
	}

	header[1] = 127
	size := make([]byte, 8)
	binary.BigEndian.PutUint64(size, uint64(payloadLen))
	header = append(header, size...)
	fullMessage := append(header, message...)
	_, err := conn.Write(fullMessage)
	return err
}
