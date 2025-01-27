package httpio

import (
	"fmt"
	"net"

	"github.com/Murilinho145SG/gouter/gouter/log"
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
		for k, v := range res.Headers {
			headers += fmt.Sprintf("%s: %s\r\n", k, v)
		}
	} else {
		log.Warn("Headers is Empty")
	}

	resStr := fmt.Sprintf("%s%s\r\n%s", statusLine, headers, string(res.Body))
	_, err := res.conn.Write([]byte(resStr))
	if err != nil {
		return err
	}

	return nil
}
