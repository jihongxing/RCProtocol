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

    fmt.Println("go-iam listening on :8083")
    _ = http.ListenAndServe(":8083", mux)
}
