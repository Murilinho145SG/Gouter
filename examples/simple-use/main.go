package main

import (
	"github.com/Murilinho145SG/gouter"
	"github.com/Murilinho145SG/gouter/httpio"
)

func main() {
	/*
		Statement to start the router to be able to read the routes
	*/
	router := gouter.NewRouter()
	router.Route("/example", func(w httpio.Writer, r *httpio.Request) {
		w.Headers().Add("Access-Control-Allow-Methods", "GET, OPTIONS")

		if r.Method == "OPTIONS" {
			w.WriteHeader(200)
			return
		}

		if r.Method != "GET" {
			w.WriteHeader(405)
			return
		}

		// For reasons of good practice and code structuring, write the header before giving a response.
		w.WriteHeader(200)
		w.Write([]byte("Hello World!"))
	})

	/*
		The connection initiator to be able to receive requests
	*/
	gouter.Run(":8080", router)
}
