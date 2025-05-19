package main

import (
	"github.com/Murilinho145SG/gouter"
)

func main() {
	r := gouter.NewRouter()

	r.Update(func(d *gouter.Doc) {
		d.Active = true       // Default true
		d.Port = "7665"       // Default 7665
		d.Addrs = "localhost" // Default localhost
	})

	r.Route("/users", func(r *gouter.Request, w *gouter.Writer) {
		// Code here...
	}, "GET").SetDescription("Get information from all users")

	r.Group("/auth", func(g *gouter.Group) {
		g.Route("/:token", func(r *gouter.Request, w *gouter.Writer) {}, "POST")
	})

	if err := gouter.Run("0.0.0.0:8080", r); err != nil {
		panic(err)
	}
}
