package httpio

type Response struct {
	Code uint
	Body []byte
	Headers
}

func NewResponse(code uint) Response {
	return Response{
		Code:    code,
		Headers: make(Headers),
	}
}
