package gouter

import (
	"errors"
	"strings"

	"github.com/Murilinho145SG/gouter/httpio"
	"github.com/Murilinho145SG/gouter/log"
)

// Handler is a function type that handles HTTP requests.
// It takes a Writer to send responses and a Request containing the request data.
type Handler func(w httpio.Writer, r *httpio.Request)

// HandlersList is a map that associates route paths with their corresponding handlers.
type HandlersList map[string]Handler

// Predefined error for when a route is not found.
var (
	ErrSearchNotFound = errors.New("this path is not registered")
)

// Router is the main router structure that manages routes and middleware.
type Router struct {
	Routes HandlersList // Map of routes and their handlers
	mw     []Middleware // List of middleware functions
}

// NewRouter creates and returns a new instance of Router with an empty route map.
func NewRouter() *Router {
	return &Router{
		Routes: make(HandlersList),
	}
}

// ParseRoute matches the request path to a registered route and returns the corresponding handler.
// It supports dynamic route parameters (e.g., `/user/:id`).
func (r *Router) ParseRoute(req *httpio.Request) Handler {
	routes := r.Routes

	// Check if the exact path exists in the routes
	if err := routes.Search(req.Path); err == nil {
		return routes.GetHandler(req.Path)
	}

	var originalPath string

	// Iterate over registered routes to find a match with dynamic parameters
	for k := range routes {
		partsReq := strings.Split(strings.Trim(req.Path, "/"), "/") // Split request path into parts
		parts := strings.Split(strings.Trim(k, "/"), "/")           // Split registered route into parts

		// Skip if the number of parts doesn't match
		if len(parts) != len(partsReq) {
			continue
		}

		var matched = true
		var currentPath string

		// Compare each part of the path
		for i := 0; i < len(parts); i++ {
			part := parts[i]
			partReq := partsReq[i]

			// Handle dynamic parameters (e.g., `:id`)
			if strings.HasPrefix(part, ":") {
				paramName := strings.TrimPrefix(part, ":") // Extract parameter name
				req.Params.Add(paramName, partReq)         // Add parameter to request
				currentPath += "/" + part                  // Build the matched path
			} else if part == partReq {
				// Handle static path parts
				if part != "" {
					currentPath += "/" + part
				}
			} else {
				// If parts don't match, mark as unmatched
				matched = false
				break
			}
		}

		// If a match is found, store the original path and break
		if matched {
			originalPath = currentPath
			break
		}
	}

	// Return the handler for the matched path
	return routes.GetHandler(originalPath)
}

// NewRoute registers a new route with its handler.
// If the route already exists, a warning is logged.
func (h HandlersList) NewRoute(path string, handler Handler) {
	if h[path] != nil {
		log.WarnSkip(2, "This path ["+path+"] already exists.")
		return
	}

	h[path] = handler
}

// Search checks if a route exists in the HandlersList.
// Returns ErrSearchNotFound if the route is not found.
func (h HandlersList) Search(path string) error {
	if h[path] == nil {
		return ErrSearchNotFound
	}

	return nil
}

// GetHandler retrieves the handler associated with a given path.
func (h HandlersList) GetHandler(path string) Handler {
	return h[path]
}

// SetDebugMode enables debug logging for the router.
func (r *Router) SetDebugMode() {
	log.DebugMode = true
}

// OnError sends an error response with the specified status code and error message.
func (r *Router) OnError(w httpio.Writer, code uint, err error) {
	w.WriteHeader(code)
	w.WriteJson(map[string]string{"error": err.Error()}, false)
}

// Route registers a new route with the router.
// Middleware is applied to the handler before registration.
func (r *Router) Route(route string, handler Handler) {
	log.InfoSkip(1, "Registering "+route)

	// Apply middleware to the handler
	for _, mw := range r.mw {
		handler = mw(handler)
	}

	// Register the route
	r.Routes.NewRoute(route, handler)
}

// Use adds middleware to the router.
func (r *Router) Use(mw Middleware) {
	r.mw = append(r.mw, mw)
}

// Group represents a group of routes with a common prefix and shared middleware.
type Group struct {
	router      *Router      // Reference to the parent router
	pathGroup   string       // Common prefix for the group
	middlewares []Middleware // Middleware specific to the group
}

// GroupFunc is a function type used to define route groups.
type GroupFunc func(g *Group)

// NewGroup creates and returns a new Group instance.
func NewGroup(router *Router, pathGroup string) *Group {
	return &Group{
		router,
		pathGroup,
		nil,
	}
}

// Middleware is a function type that wraps a Handler to provide additional functionality.
type Middleware func(Handler) Handler

// UseGroup adds middleware to the group.
func (g *Group) UseGroup(mw Middleware) {
	g.middlewares = append(g.middlewares, mw)
}

// Group creates a new route group with a common prefix and optional middleware.
func (r *Router) Group(pathGroup string, handler GroupFunc) {
	log.InfoSkip(1, "Registering Group "+pathGroup)
	g := NewGroup(r, pathGroup)
	handler(g)
}

// Route registers a route within the group.
// Middleware specific to the group is applied to the handler.
func (g *Group) Route(route string, handler Handler) {
	log.InfoSkip(3, "Registering "+g.pathGroup+route)

	// Apply group middleware to the handler
	for _, mw := range g.middlewares {
		handler = mw(handler)
	}

	// Register the route with the group prefix
	g.router.Route(g.pathGroup+route, handler)
}
