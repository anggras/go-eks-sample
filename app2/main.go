package main

import (
    "fmt"
    "net/http"
)

func main() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        fmt.Fprintf(w, "Hello, you've requested: %s\n", r.URL.Path)
        fmt.Fprintf(w, "APP 2\n")
    })

    http.ListenAndServe(":8000", nil)
}