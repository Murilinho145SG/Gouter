package gouter

import (
	"github.com/Murilinho145SG/gouter/gouter/httpio"
	"github.com/Murilinho145SG/gouter/gouter/log"
)

type Handler func(w httpio.Writer, r *httpio.Request)
type HandlersList map[string]Handler

type Router struct {
	Routes    HandlersList
	DebugMode bool
}

func NewRouter() *Router {
	return &Router{
		Routes:    make(HandlersList),
		DebugMode: false,
	}
}

func (r *Router) SetDebugMode() {
	r.DebugMode = true
}

func (r *Router) Route(route string, handler Handler) {
	log.Info("Registering " + route)
	r.Routes[route] = handler
}
