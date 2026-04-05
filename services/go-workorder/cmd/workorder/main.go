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

    fmt.Println("go-workorder listening on :8085")
    _ = http.ListenAndServe(":8085", mux)
}
