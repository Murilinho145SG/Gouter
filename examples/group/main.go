package main

import "github.com/Murilinho145SG/gouter"

func main() {
	r := gouter.NewRouter()

	r.Group("/user", func(g *gouter.Group) {
		g.Route("/features", func(r *gouter.Request, w *gouter.Writer) {
			//Code here...
		})

		g.Route("/infos", func(r *gouter.Request, w *gouter.Writer) {
			//Code here...
		})

		/*
			Outputs:
				/user/features
				/user/infos
		*/
	})

	if err := gouter.Run("0.0.0.0:8080", r); err != nil {
		panic(err)
	}
}
