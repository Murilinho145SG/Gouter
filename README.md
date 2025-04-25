# 🚀 Gouter - Lightweight HTTP Router for Go

[![Go Report Card](https://goreportcard.com/badge/github.com/Murilinho145SG/gouter)](https://goreportcard.com/report/github.com/Murilinho145SG/gouter)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](https://opensource.org/licenses/MIT)

**Gouter** is a minimalist yet powerful HTTP router for Go, designed for building high-performance web applications and APIs. Inspired by Express.js and standard lib from golang, it combines simplicity with essential features for modern web development.

## ✨ Features

- 🛣️ **Intuitive Routing** with dynamic parameters
- 🔌 **Middleware Support** for modular code
- 🧩 **Route Grouping** with shared prefixes and middleware
- 📡 **HTTPS/HTTP2 Support** out of the box
- 📊 **Automatic Body Parsing** with intelligent buffering
- 📝 **Structured Logging** with color-coded output
- 🚦 **Error Handling** with JSON error responses
- ⚡ **High Performance** with concurrent connection handling

## 📦 Installation

```bash
go get github.com/Murilinho145SG/gouter
```

🚀 Quick Start

```go
package main

import (
	"github.com/Murilinho145SG/gouter"
	"github.com/Murilinho145SG/gouter/httpio"
)

func main() {
	r := gouter.NewRouter()
	
	r.Route("/", func(w httpio.Writer, req *httpio.Request) {
		w.WriteJson(map[string]string{"message": "Welcome to Gouter!"}, false)
	})

	r.Route("/user/:name", func(w httpio.Writer, req *httpio.Request) {
		name, _ := req.Params.Get("name")
		w.WriteJson(map[string]string{"hello": name}, false)
	})

	gouter.Run(":8080", r)
}
```

🛠️ Advanced Usage

Middleware Example
```go
func Logger(next gouter.Handler) gouter.Handler {
	return func(w httpio.Writer, r *httpio.Request) {
		log.Info("Request:", r.Method, r.Path)
		next(w, r)
	}
}

func main() {
	r := gouter.NewRouter()
	r.Use(Logger)
	
	// Routes...
}
```

Route Groups

```go
func main() {
	r := gouter.NewRouter()
	
	api := r.Group("/api")
	api.Use(JWTAuth)
	
	api.Route("/users", UsersHandler)
	api.Route("/posts", PostsHandler)
}
```

Dynamic Routes

```go
r.Route("/product/:category/:id", func(w httpio.Writer, r *httpio.Request) {
	category, _ := r.Params.Get("category")
	id, _ := r.Params.Get("id")
	// Handle request...
})
```

Error Handling
```go
r.OnError = func(w httpio.Writer, code uint, err error) {
	w.WriteHeader(code)
	w.WriteJson(map[string]string{
		"error": err.Error(),
		"code":  strconv.Itoa(int(code)),
	}, false)
}
```

HTTPS Support

```go
err := gouter.RunTLS(
	":443",
	r,
	"cert.pem",
	"key.pem",
)
```

📊 Logging
Enable debug mode for detailed request logging:

```go
r.SetDebugMode()
```

🤝 Contributing
We welcome contributions! Please see our Contribution Guidelines for details.

📄 License
MIT License - See [LICENSE](https://github.com/Murilinho145SG/Gouter/blob/main/LICENSE) for details.

Happy Routing! 🎉 Built with ❤️ by Murilinho145
