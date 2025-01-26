package httpio

type Response struct {
	Code uint
	Body []byte
	Headers
}

func NewResponse() Response {
	return Response{
		Headers: make(Headers),
	}
}
