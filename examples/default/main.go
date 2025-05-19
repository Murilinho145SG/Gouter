package main

import "github.com/Murilinho145SG/gouter"

func main() {
	r := gouter.NewRouter()
	
	r.Route("/", func(r *gouter.Request, w *gouter.Writer) {
		w.Write([]byte("Hello World!"))
	})
	
	if err := gouter.Run("0.0.0.0:8080", r); err != nil {
		panic(err)
	}
}