package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	port := ":9001"
	name := "Backend 1"

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s %s from %s", name, r.Method, r.URL.Path, r.RemoteAddr)
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintln(w, "Hello from Backend 1!")
	})

	log.Printf("%s starting on port %s...", name, port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("[%s] Failed to start server: %v", name, err)
	}
}