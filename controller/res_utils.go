package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

func show404(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintln(w, "404 not found")
}

func sendHTML(w http.ResponseWriter, content string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(content))
}

func sendJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	var b bytes.Buffer

	if err := json.NewEncoder(&b).Encode(data); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, "json encoding error:", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	fmt.Fprintln(w, b.String())
}
