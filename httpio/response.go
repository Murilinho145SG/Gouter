package httpio

import (
	"fmt"
	"net"
	"net/textproto"
	"strconv"

	"github.com/Murilinho145SG/gouter/log"
)

type Response struct {
	Code uint
	Body []byte
	Headers
	conn net.Conn
}

func NewResponse(conn net.Conn) Response {
	return Response{
		Headers: make(Headers),
		conn:    conn,
	}
}

func (res *Response) Write() error {
	var statusLine string
	if res.Code == 0 {
		statusLine = fmt.Sprintf("HTTP/1.1 %d\r\n", 404)
		log.Warn("No have response code")
	} else {
		statusLine = fmt.Sprintf("HTTP/1.1 %d\r\n", res.Code)
	}

	var headers string
	if len(res.Headers) != 0 {
		if len(res.Body) > 0 {
			err := res.Headers.Add("Content-Length", strconv.Itoa(len(res.Body)))
			if err != nil {
				log.WarnSkip(1, "You do not need to declare the body size. The size is already declared automatically")
				res.Headers.Del("Content-Length")
				res.Headers.Add("Content-Length", strconv.Itoa(len(res.Body)))
			}
		}

		for k, v := range res.Headers {
			value := textproto.TrimString(v)
			headers += fmt.Sprintf("%s: %s\r\n", k, value)
		}
	} else {
		if len(res.Body) > 0 {
			res.Headers.Add("Content-Length", strconv.Itoa(len(res.Body)))
			log.WarnSkip(1, "Headers is Empty")
		}
	}

	resStr := fmt.Sprintf("%s%s\r\n%s", statusLine, headers, string(res.Body))

	log.DebugSkip(1, resStr)

	_, err := res.conn.Write([]byte(resStr))
	if err != nil {
		return err
	}

	return nil
}
