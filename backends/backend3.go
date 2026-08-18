package main

import (
    "fmt"
    "net/http"
)

func main() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintln(w, "Hello from Backend 3!")
    })

    fmt.Println("Backend 3 running on port 9003")
    http.ListenAndServe(":9003", nil)
}