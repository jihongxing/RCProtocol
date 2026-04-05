package main

import (
    "fmt"
    "net/http"
)

func main() {
    mux := http.NewServeMux()
    mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
        _, _ = fmt.Fprint(w, "ok")
    })
    mux.HandleFunc("/console/dashboard", func(w http.ResponseWriter, r *http.Request) {
        _, _ = fmt.Fprint(w, "dashboard placeholder")
    })

    fmt.Println("go-bff listening on :8082")
    _ = http.ListenAndServe(":8082", mux)
}
