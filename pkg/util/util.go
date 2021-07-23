package util

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	nodev1alpha1 "github.com/ibrokethecloud/harvester-tink-operator/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"

	"github.com/pkg/errors"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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
func FetchServerURL(apiClient client.Client, node *nodev1alpha1.Register) (url string, err error) {

	ok, err := DoesSettingExist(apiClient)
	if err != nil {
		return url, err
	}

	if ok {
		serverURL := &unstructured.Unstructured{
			Object: map[string]interface{}{
				"apiVersion": "harvesterhci.io/v1beta1",
				"kind":       "Setting",
			},
		}

		err = apiClient.Get(context.TODO(), types.NamespacedName{Name: "server-url", Namespace: ""}, serverURL)

		if err != nil {
			return url, errors.Wrap(err, "error fetching server-url")
		}

		regoURL, ok := serverURL.Object["value"]
		if !ok {
			return url, fmt.Errorf("server-url value is not set")
		}

		urlArr := strings.Split(regoURL.(string), ":")

		url = strings.Join(urlArr[:len(urlArr)-1], ":") + ":6443"

		return url, nil
	}

	// multi cluster mode running
	// need to fetch cluster and then identify object
	clusterName, ok := node.Labels["clusterName"]
	if !ok {
		return url, fmt.Errorf("error querying clusterName label on node")
	}
	nodeLabels, err := labels.Parse("clusterName=" + clusterName)
	list := &nodev1alpha1.RegisterList{}
	err = apiClient.List(context.TODO(), list, &client.ListOptions{LabelSelector: nodeLabels})
	if err != nil {
		return url, errors.Wrap(err, "error fetching cluster member list")
	}

	for _, node := range list.Items {
		if leader, ok := node.Labels["leader"]; ok && leader == "true" {
			url = "https://" + node.Spec.Address + ":6443"
			return url, nil
		}
	}

	// got till here.. which means we didnt find a leader
	return url, fmt.Errorf("no leader found in cluster node list")
}

func DoesSettingExist(client client.Client) (ok bool, err error) {
	crdList := &apiextensions.CustomResourceDefinitionList{}
	err = client.List(context.TODO(), crdList)
	if err != nil {
		return ok, errors.Wrap(err, "error listing CRD's")
	}

	for _, crd := range crdList.Items {
		if crd.Name == "settings.harvesterhci.io" {
			ok = true
		}
	}
	return ok, nil
}
