package util

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type HTTPMessage struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func ReturnHTTPMessage(w http.ResponseWriter, r *http.Request, httpStatus int, messageType string, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)

	err := HTTPMessage{
		Status:  strconv.Itoa(httpStatus),
		Message: message,
		Type:    messageType,
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(err)
}

type HTTPContent struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Content []byte `json:"content"`
}

func ReturnHTTPContent(w http.ResponseWriter, r *http.Request, httpStatus int, messageType string, content []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)

	err := HTTPContent{
		Status:  strconv.Itoa(httpStatus),
		Content: content,
		Type:    messageType,
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	enc.Encode(err)
}

func ReturnHTTPRaw(w http.ResponseWriter, r *http.Request, content string) {
	fmt.Fprintf(w, "%s", content)
}

// helper to find registration url //
func FetchServerURL(client client.Client) (url string, err error) {
	serverURL := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "harvesterhci.io/v1beta1",
			"kind":       "Setting",
		},
	}

	err = client.Get(context.TODO(), types.NamespacedName{Name: "server-url", Namespace: ""}, serverURL)

	if err != nil {
		return url, errors.Wrap(err, "error fetching server-url")
	}

	regoURL, ok := serverURL.Object["value"]
	if !ok {
		return url, fmt.Errorf("server-url value is not set")
	}

	url = regoURL.(string)
	return url, nil
}
