package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

type logresponse struct {
	http.ResponseWriter
	statusCode    int
	contentLength int
}

func (lr *logresponse) Write(b []byte) (int, error) {
	size, err := lr.ResponseWriter.Write(b)
	lr.contentLength += size
	return size, err
}

func (lr *logresponse) WriteHeader(statusCode int) {
	lr.ResponseWriter.WriteHeader(statusCode)
	lr.statusCode = statusCode
}

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wrapper := &logresponse{ResponseWriter: w}

		next.ServeHTTP(wrapper, r)
		duration := time.Since(start)
		ctype := parseContentType(wrapper.Header().Get("Content-Type"))

		log.Printf(
			"%s %s -- %d %s [%s of %s, in %s]",
			r.Method, r.URL.Path,
			wrapper.statusCode, http.StatusText(wrapper.statusCode),
			humanizeSize(wrapper.contentLength), ctype, duration.String(),
		)
	})
}

func parseContentType(str string) string {
	if pos := strings.IndexByte(str, ';'); pos >= 0 {
		return str[:pos]
	}
	return str
}

func humanizeSize(b int) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d bytes", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cb", float64(b)/float64(div), "kmgtpe"[exp])
}
