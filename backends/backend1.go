package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
)

func main() {
	port := ":9001"
	name := "Backend 1"

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("[%s] %s %s from %s", name, r.Method, r.URL.Path, r.RemoteAddr)
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}
		defer r.Body.Close()

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "Backend 1 received: %s (method: %s)", string(body), r.Method)
	})

	log.Printf("%s starting on port %s...", name, port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("[%s] Failed to start server: %v", name, err)
	}
}
