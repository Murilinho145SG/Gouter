/*
Package gouter provides HTTP routing capabilities with middleware support and route grouping.

Features:
- Parameterized routes (e.g., /users/:id)
- Route groups with prefixing
- Middleware chaining
- Route documentation generation
- Wildcard route support
*/
package gouter

import (
	"errors"
	"strings"

	"github.com/Murilinho145SG/gouter/log"
)

// Handler defines the function signature for request handlers
type Handler func(r *Request, w *Writer)

// handlerList maps route paths to their corresponding handlers
type handlerList map[string]Handler

// Middleware defines the function signature for middleware processors
type Middleware func(handler Handler) Handler

// router manages routes, middleware, and documentation
type Router struct {
	handlerList handlerList // Map of registered routes
	mws         []Middleware // List of global middlewares
	docs        []*RouteInfo // Route documentation store
}

// RouteInfo contains documentation metadata for a route
type RouteInfo struct {
	Method      string      // HTTP method (GET, POST, etc.)
	Path        string      // Route path pattern
	Description string      // Human-readable description
	Parameters  []ParamInfo // List of path parameters
}

// ParamInfo describes a path parameter
type ParamInfo struct {
	Name        string // Parameter name (e.g., "id")
	Type        string // Expected data type
	Description string // Parameter description
}

// SetDescription sets the route description and returns modified RouteInfo
func (r *RouteInfo) SetDescription(desc string) *RouteInfo {
	r.Description = desc
	return r
}

// SetParam updates parameter metadata and returns modified RouteInfo
func (r *RouteInfo) SetParam(paramName, ty, desc string) *RouteInfo {
	for i, param := range r.Parameters {
		if param.Name == paramName {
			r.Parameters[i].Type = ty
			r.Parameters[i].Description = desc
		}
	}
	return r
}

// NewRouter creates and returns a new router instance
func NewRouter() *Router {
	return &Router{
		handlerList: make(handlerList),
	}
}

// parseRoute matches incoming requests to registered routes
// Returns the appropriate handler or nil if no match found
func (r *Router) parseRoute(req *Request) Handler {
	if req == nil {
		return nil
	}

	routes := r.handlerList

	// Check for exact match
	if err := routes.hasRoute(req.Path); err == nil {
		return routes.getHandler(req.Path)
	}

	var originalPath string

	// Check for wildcard and parameterized routes
	for k := range routes {
		// Handle wildcard routes (e.g., /static/*)
		if strings.HasSuffix(k, "/*") {
			baseRoute := strings.TrimSuffix(k, "/*")
			if strings.HasPrefix(req.Path, baseRoute+"/") {
				return routes.getHandler(k)
			}
		}

		// Split path segments for parameter matching
		partsReq := strings.Split(strings.Trim(req.Path, "/"), "/")
		parts := strings.Split(strings.Trim(k, "/"), "/")

		if len(parts) != len(partsReq) {
			continue
		}

		var matched = true
		var currentPath string

		// Match path segments
		for i := 0; i < len(parts); i++ {
			part := parts[i]
			partReq := partsReq[i]

			// Handle parameter segments (e.g., :id)
			if strings.HasPrefix(part, ":") {
				paramName := strings.TrimPrefix(part, ":")
				req.Params.add(paramName, partReq)
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

		if matched && len(parts) == len(partsReq) {
			originalPath = currentPath
			break
		}
	}

	return routes.getHandler(originalPath)
}

// Route registers a new handler for a specific path
// methods: Optional HTTP method specification (defaults to GET)
// Returns RouteInfo for documentation purposes
func (r *Router) Route(path string, handler Handler, methods ...string) *RouteInfo {
	// Check for existing route
	if r.handlerList[path] != nil {
		log.WarnE(2, "This path ["+path+"] already exists.")
		return nil
	}

	// Apply middleware chain
	for _, mw := range r.mws {
		handler = mw(handler)
	}

	r.handlerList[path] = handler

	// Create documentation entry
	doc := RouteInfo{
		Path:   path,
		Method: "GET", // Default method
	}

	if len(methods) > 0 {
		doc.Method = methods[0]
	}

	doc.Parameters = []ParamInfo{}

	// Extract parameters from path
	parts := strings.Split(path, "/")
	for _, part := range parts {
		if strings.HasPrefix(part, ":") {
			paramName := strings.TrimPrefix(part, ":")
			doc.Parameters = append(doc.Parameters, ParamInfo{
				Name: paramName,
			})
		}
	}

	r.docs = append(r.docs, &doc)

	return &doc
}

// Use adds middleware to the global middleware chain
func (r *Router) Use(mw Middleware) {
	r.mws = append(r.mws, mw)
}

// hasRoute checks if a path exists in the handler list
func (h handlerList) hasRoute(path string) error {
	if h[path] == nil {
		return errors.New("path is not founded")
	}
	return nil
}

// getHandler retrieves the handler for a specific path
func (h handlerList) getHandler(path string) Handler {
	return h[path]
}

// Group creates a route group with common configuration
func (r *Router) Group(path string, handler GroupFunc) {
	handler(newGroup(r, path))
}

// Group represents a set of routes with shared configuration
type Group struct {
	router    *Router      // Parent router
	pathGroup string       // Group path prefix
	mw        []Middleware // Group-specific middleware
}

// GroupFunc defines the function signature for group configuration
type GroupFunc func(g *Group)

// newGroup creates a new route group instance
func newGroup(router *Router, pathGroup string) *Group {
	return &Group{
		router:    router,
		pathGroup: pathGroup,
	}
}

// Route registers a route within the group
func (g *Group) Route(path string, handler Handler) {
	// Apply group middleware
	for _, mw := range g.mw {
		handler = mw(handler)
	}

	// Register route with group prefix
	g.router.Route(g.pathGroup+path, handler)
}

// Use adds middleware to the group's middleware chain
func (g *Group) Use(mw Middleware) {
	g.mw = append(g.mw, mw)
}