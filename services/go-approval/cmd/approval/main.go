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

    fmt.Println("go-approval listening on :8084")
    _ = http.ListenAndServe(":8084", mux)
}
