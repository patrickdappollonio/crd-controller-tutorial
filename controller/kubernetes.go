package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

const (
	pathToCertificate = "/var/run/secrets/kubernetes.io/serviceaccount/ca.crt"
	pathToToken       = "/var/run/secrets/kubernetes.io/serviceaccount/token"
	pathToCurrentNS   = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"

	defaultNamespace = "default"
)

func doRequest(method, endpoint, token string, body io.Reader, contents interface{}) error {
	req, err := http.NewRequest(method, endpoint, body)
	if err != nil {
		return fmt.Errorf("unable to create https request to %q: %w", endpoint, err)
	}

	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client, err := getClient()
	if err != nil {
		return fmt.Errorf("unable to obtain https client: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("unable to perform https request to %q: %w", endpoint, err)
	}

	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// do nothing

	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("unable to perform request: the request was not authorized\n%s", stringifyReader(resp.Body))

	case http.StatusNotFound:
		return fmt.Errorf("resource not found")

	default:
		return fmt.Errorf("unexpected status code %q received:\n%s", resp.Status, stringifyReader(resp.Body))
	}

	if err := json.NewDecoder(resp.Body).Decode(contents); err != nil {
		return fmt.Errorf("unable to decode JSON output for %q: %w", endpoint, err)
	}

	return nil
}

func getClient() (*http.Client, error) {
	if !isKubernetes() {
		return http.DefaultClient, nil
	}

	certAuthority, err := os.ReadFile(pathToCertificate)
	if err != nil {
		return nil, fmt.Errorf("unable to read file %q: %w", pathToCertificate, err)
	}

	allCAs, _ := x509.SystemCertPool()
	if allCAs == nil {
		allCAs = x509.NewCertPool()
	}

	allCAs.AppendCertsFromPEM(certAuthority)

	tlsConfig := &tls.Config{RootCAs: allCAs}

	client := http.DefaultClient
	client.Transport = &http.Transport{TLSClientConfig: tlsConfig}

	return client, nil
}

func getToken() (string, error) {
	if !isKubernetes() {
		return "", nil
	}

	tokenBytes, err := os.ReadFile(pathToToken)
	if err != nil {
		return "", fmt.Errorf("unable to read file %q: %w", pathToToken, err)
	}

	return string(tokenBytes), nil
}

func getCurrentNamespace() (string, error) {
	if !isKubernetes() {
		return defaultNamespace, nil
	}

	nsbytes, err := os.ReadFile(pathToCurrentNS)
	if err != nil {
		return "", fmt.Errorf("unable to read file %q: %w", pathToCurrentNS, err)
	}

	return string(nsbytes), nil
}
