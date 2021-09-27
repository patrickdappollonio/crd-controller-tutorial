package main

import (
	"bytes"
	"io"
	"os"
	"strings"
)

func envdefault(key, defval string) string {
	if s := strings.TrimSpace(os.Getenv(key)); s != "" {
		return s
	}

	return defval
}
func stringifyReader(in io.Reader) string {
	var b bytes.Buffer

	if _, err := io.Copy(&b, in); err != nil {
		return ""
	}

	return b.String()
}

func isKubernetes() bool {
	return os.Getenv("KUBERNETES_SERVICE_HOST") != ""
}

func removeTrailingSlash(str string) string {
	return strings.TrimSuffix(str, "/")
}
