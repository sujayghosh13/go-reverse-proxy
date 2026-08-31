//go:build ignore

package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

func main() {
	port := ":9002"
	name := "Backend 2"

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s %s from %s", name, r.Method, r.URL.Path, r.RemoteAddr)
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		_ = body

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "Hello from Backend 2!")
	})

	log.Printf("%s starting on port %s...", name, port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("[%s] Failed to start server: %v", name, err)
	}
}
