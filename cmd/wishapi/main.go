package main

import (
	"log"
	"net/http"
)

func testHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello http"))
}

func main() {
	if err := http.ListenAndServe("0.0.0.0:3000", http.HandlerFunc(testHandler)); err != nil {
		log.Fatal(err)
	}
}
