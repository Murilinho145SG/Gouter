package gouter

import (
	"os"

	"github.com/Murilinho145SG/gouter/httpio"
	"github.com/Murilinho145SG/gouter/log"
	"github.com/Murilinho145SG/gouter/tester"
)

type Handler func(w httpio.Writer, r *httpio.Request)
type HandlersList map[string]Handler

type Router struct {
	Routes    HandlersList
	DebugMode bool
}

func NewRouter(debug bool) *Router {
	if debug {
		err := tester.StartTest()
		if err != nil {
			log.Error("Error when testing the application", err)
			os.Exit(1)
		}
	}

	return &Router{
		Routes:    make(HandlersList),
		DebugMode: debug,
	}
}

func (r *Router) Route(route string, handler Handler) {
	log.Info("Registering " + route)
	r.Routes[route] = handler
}
