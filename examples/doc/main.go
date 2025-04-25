package main

import "github.com/Murilinho145SG/gouter"

func main() {
	r := gouter.NewRouter()
	
	r.Route("/users", func(r *gouter.Request, w *gouter.Writer) {
		// Code here...
	}, "GET").SetDescription("Get information from all users")
	
	if err := gouter.Run("0.0.0.0", r); err != nil {
		panic(err)
	}
}