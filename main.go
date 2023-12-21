package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "<h1>Welcome to App Runner</h1>")
	})
	fmt.Println("Starting the server on :8080...")
	http.ListenAndServe(":8080", nil)
}
