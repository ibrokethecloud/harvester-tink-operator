package util

import (
	"context"
	"encoding/json"
	"fmt"
	nodev1alpha1 "github.com/ibrokethecloud/harvester-tink-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"net/http"
	"os"
	"strconv"

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
	namespace := os.Getenv("namespace")
	service := &corev1.Service{}
	err = client.Get(context.TODO(), types.NamespacedName{Name: "harvester-tink-operator", Namespace: namespace}, service)
	if err != nil {
		return url, err
	}

	address := os.Getenv("PUBLIC_IP")

	url = fmt.Sprintf("http://%s:%s", address, nodev1alpha1.DefaultConfigURLPort)

	return url, nil
}

// helper to find harvester version
func FindHarvesterVersion(client client.Client) (version string, err error) {
	versionObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "harvesterhci.io/v1beta1",
			"kind":       "Setting",
		},
	}

	err = client.Get(context.TODO(), types.NamespacedName{Name: "server-version", Namespace: ""}, versionObj)
	if err != nil {
		return version, err
	}

	version = versionObj.Object["value"].(string)
	return version, err
}
