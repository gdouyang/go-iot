package httpserver

import (
	"fmt"
	"net/http"
)

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "hello world")
}

func ServerStart() {
	http.HandleFunc("/", indexHandler)
	http.ListenAndServe(":8000", nil)
}
