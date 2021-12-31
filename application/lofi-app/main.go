package main

import (
	_ "embed"
	"io"
	"log"
	"net/http"
)

//go:embed lofi.gif
var font []byte

func main() {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("request:", r.URL.Path)
		w.Header().Set("Content-Type", "image/gif")
		if _, err := io.WriteString(w, string(font)); err != nil {
			log.Fatal(err)
		}
	})

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if _, err := io.WriteString(w, "OK"); err != nil {
			log.Fatal(err)
		}

	})

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal(err)
	}

}
