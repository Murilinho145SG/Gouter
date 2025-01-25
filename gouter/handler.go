package gouter

import "github.com/Murilinho145SG/gouter/gouter/httpio"

type Handler func(w httpio.Writer, r *httpio.Request)
type HandlersList map[string]Handler

type Router struct {
	Routes HandlersList
}

func NewRouter() *Router {
	return &Router{
		Routes: make(HandlersList),
	}
}

func (r *Router) Route(route string, handler Handler) {
	r.Routes[route] = handler
}
