package router

import (
	"encoding/json"
	"log"
	"net/http"

	"gorm.io/gorm"
)

type ApiHandler func(ctx *Context)

type Context struct {
	R     *http.Request
	W     http.ResponseWriter
	Route string
}

func (c *Context) ReadJson(v any) error {
	return json.NewDecoder(c.R.Body).Decode(v)
}

func (c *Context) WriteJson(v any) error {
	return json.NewEncoder(c.W).Encode(v)
}

func (c *Context) WriteError(status int, err error) error {
	c.W.WriteHeader(status)
	mapErr := map[string]string{
		"error": err.Error(),
	}
	return json.NewEncoder(c.W).Encode(mapErr)
}

func (c *Context) WriteStatus(status int) {
	c.W.WriteHeader(status)
}

func NewContext(r *http.Request, w http.ResponseWriter, route string) *Context {
	return &Context{
		R:     r,
		W:     w,
		Route: route,
	}
}

func Post(route string, handler ApiHandler) {
	log.Println("Registering POST route", route)

	http.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx := NewContext(r, w, route)
		handler(ctx)
	})
}

func Get(route string, handler ApiHandler) {
	log.Println("Registering GET route", route)

	http.HandleFunc(route, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		ctx := NewContext(r, w, route)
		handler(ctx)
	})
}

func ListenAndServe(port string) error {
	return http.ListenAndServe(port, nil)
}

func ListenAndServeTLS(port string, certFile string, keyFile string) error {
	return http.ListenAndServeTLS(port, certFile, keyFile, nil)
}
