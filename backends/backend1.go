package main

import (
    "fmt"
    "net/http"
)

func main() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "Hello from Backend 1!")
    })

    fmt.Println("Backend 1 running on port 9001")
    http.ListenAndServe(":9001", nil)
}