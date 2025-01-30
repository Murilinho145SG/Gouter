package gouter

import (
	"errors"
	"strings"

	"github.com/Murilinho145SG/gouter/httpio"
	"github.com/Murilinho145SG/gouter/log"
)

type Handler func(w httpio.Writer, r *httpio.Request)
type HandlersList map[string]Handler

var (
	ErrSearchNotFound = errors.New("this path is not registered")
)

type Router struct {
	Routes HandlersList
	mw     []Middleware
}

func NewRouter() *Router {
	return &Router{
		Routes: make(HandlersList),
	}
}

func (r *Router) ParseRoute(req *httpio.Request) Handler {
	routes := r.Routes
	if err := routes.Search(req.Path); err == nil {
		return routes.GetHandler(req.Path)
	}

	var originalPath string

	for k := range routes {
		partsReq := strings.Split(strings.Trim(req.Path, "/"), "/")
		parts := strings.Split(strings.Trim(k, "/"), "/")

		if len(parts) != len(partsReq) {
			continue
		}

		var matched = true
		var currentPath string

		for i := 0; i < len(parts); i++ {
			part := parts[i]
			partReq := partsReq[i]

			if strings.HasPrefix(part, ":") {
				paramName := strings.TrimPrefix(part, ":")

				req.Params.Add(paramName, partReq)
				currentPath += "/" + part
			} else if part == partReq {
				if part != "" {
					currentPath += "/" + part
				}
			} else {
				matched = false
				break
			}
		}

		if matched {
			originalPath = currentPath
			break
		}
	}

	return routes.GetHandler(originalPath)
}

func (h HandlersList) NewRoute(path string, handler Handler) {
	if h[path] != nil {
		log.WarnSkip(2, "This path ["+path+"] already exists.")
		return
	}

	h[path] = handler
}

func (h HandlersList) Search(path string) error {
	if h[path] == nil {
		return ErrSearchNotFound
	}

	return nil
}

func (h HandlersList) GetHandler(path string) Handler {
	return h[path]
}

func (r *Router) SetDebugMode() {
	log.DebugMode = true
}

func (r *Router) OnError(w httpio.Writer, code uint, err error) {
	w.WriteHeader(code)
	w.WriteJson(map[string]string{"error": err.Error()}, false)
}

func (r *Router) Route(route string, handler Handler) {
	log.InfoSkip(1, "Registering "+route)

	for _, mw := range r.mw {
		handler = mw(handler)
	}

	r.Routes.NewRoute(route, handler)
}

func (r *Router) Use(mw Middleware) {
	r.mw = append(r.mw, mw)
}

type Group struct {
	router      *Router
	pathGroup   string
	middlewares []Middleware
}

type GroupFunc func(g *Group)

func NewGroup(router *Router, pathGroup string) *Group {
	return &Group{
		router,
		pathGroup,
		nil,
	}
}

type Middleware func(Handler) Handler

func (g *Group) UseGroup(mw Middleware) {
	g.middlewares = append(g.middlewares, mw)
}

func (r *Router) Group(pathGroup string, handler GroupFunc) {
	log.InfoSkip(1, "Registering Group "+pathGroup)
	g := NewGroup(r, pathGroup)
	handler(g)
}

func (g *Group) Route(route string, handler Handler) {
	log.InfoSkip(3, "Registering "+g.pathGroup+route)

	for _, mw := range g.middlewares {
		handler = mw(handler)
	}

	g.router.Routes.NewRoute(g.pathGroup+route, handler)
}
