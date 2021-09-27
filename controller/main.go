package main

import (
	"log"
	"net/http"
	"strings"

	_ "embed"
)

const (
	resourcePath = "/apis/{apiVersion}/namespaces/{namespace}/{pluralName}?limit={limit}"
)

var (
	kubeAPIServerHost  = envdefault("KUBERNETES_API_HOST", "https://kubernetes.default.svc")
	resourceAPIVersion = envdefault("KUBERNETES_API_VERSION", "patrickdap.com/v1")
	resourceNamePlural = envdefault("KUBERNETES_RESOURCE_PLURAL", "todos")
	resourceQueryLimit = envdefault("KUBERNETES_RESOURCE_LIMIT", "500")
	controllerPort     = envdefault("KUBERNETES_CONTROLLER_PORT", "8080")
)

//go:embed homepage.tmpl
var homepageCode string

func main() {
	kubeAPIServerHost = removeTrailingSlash(kubeAPIServerHost)

	serviceAccountToken, err := getToken()
	if err != nil {
		log.Fatalf("unable to retrieve service account token for pod: %s", err.Error())
	}

	currentNamespace, err := getCurrentNamespace()
	if err != nil {
		log.Fatalf("unable to retrieve current namespace for pod: %s", err.Error())
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/", showHomeOrNotFound)
	mux.HandleFunc("/api/todos", showTodosForNamespace(kubeAPIServerHost, currentNamespace, serviceAccountToken))

	srv := &http.Server{
		Addr:    ":" + controllerPort,
		Handler: logRequest(mux),
	}

	log.Printf("Starting server. Listening on %s", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Unable to start server on %s: %s", srv.Addr, err.Error())
	}
}

func showHomeOrNotFound(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		show404(w)
		return
	}

	sendHTML(w, homepageCode)
}

func showTodosForNamespace(kubeApi, namespace, token string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		reqpath := strings.NewReplacer(
			"{apiVersion}", resourceAPIVersion,
			"{namespace}", namespace,
			"{pluralName}", resourceNamePlural,
			"{limit}", resourceQueryLimit,
		).Replace(resourcePath)

		var data todoData

		if err := doRequest(http.MethodGet, kubeApi+reqpath, token, nil, &data); err != nil {
			sendJSON(w, http.StatusInternalServerError, map[string]string{
				"error": err.Error(),
			})
			return
		}

		sendJSON(w, http.StatusOK, map[string]interface{}{
			"items":     specToMap(data),
			"namespace": namespace,
		})
	}
}

type todoData struct {
	Items []struct {
		Spec struct {
			Name string `json:"name"`
		} `json:"spec"`
	}
}

func specToMap(items todoData) []string {
	m := make([]string, 0, len(items.Items))
	for _, v := range items.Items {
		m = append(m, v.Spec.Name)
	}
	return m
}
