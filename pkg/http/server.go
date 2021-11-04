package http

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/ghodss/yaml"
	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	"github.com/ibrokethecloud/harvester-tink-operator/api/v1alpha1"
	installer "github.com/ibrokethecloud/harvester-tink-operator/pkg/installer"
	"github.com/ibrokethecloud/harvester-tink-operator/pkg/util"
	apierror "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ConfigServer struct {
	client.Client
	Log logr.Logger
}

func (c *ConfigServer) SetupRoutes(r *mux.Router) {
	r.HandleFunc("/config/{uuid}", c.getConfig).Methods("GET")
	c.Log.Info("adding config route")
}

func (c *ConfigServer) getConfig(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	configUUID, ok := vars["uuid"]
	if !ok {
		util.ReturnHTTPMessage(w, r, 500, "error", "no config uuid passed in")
		return
	}
	c.Log.Info("serving request " + configUUID)
	nodeList := &v1alpha1.RegisterList{}
	nodeLabels, err := labels.Parse("uuid=" + configUUID)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "label parsing error")
	}
	opts := &client.ListOptions{LabelSelector: nodeLabels}
	err = c.List(context.Background(), nodeList, opts)
	if err != nil {
		if apierror.IsNotFound(err) {
			util.ReturnHTTPMessage(w, r, 404, "error", "no config found")
			return
		} else {
			util.ReturnHTTPMessage(w, r, 500, "error", "internal error")
			return
		}
	}

	if len(nodeList.Items) != 1 {
		util.ReturnHTTPMessage(w, r, 500, "error", "object lookup error")
		return
	}

	node := nodeList.Items[0]

	// check if node is already registered in which case disable serving the url //
	if _, ok := node.Labels["nodeReady"]; ok {
		util.ReturnHTTPMessage(w, r, 200, "info", "node already processed")
		return
	}
	serverURL, err := util.FetchServerURL(c.Client)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "server-url fetch error")
		return
	}

	serverURLArr := strings.Split(serverURL, ":")
	os := installer.OS{
		Hostname: node.Name,
	}

	if len(node.Spec.SSHAuthorizedKeys) != 0 {
		os.SSHAuthorizedKeys = node.Spec.SSHAuthorizedKeys
	}

	if len(node.Spec.Password) != 0 {
		os.Password = node.Spec.Password
	} else {
		os.Password = node.Name
	}

	if len(node.Spec.NTPServers) != 0 {
		os.NTPServers = node.Spec.NTPServers
	}

	if len(node.Spec.DNSNameservers) != 0 {
		os.DNSNameservers = node.Spec.NTPServers
	}

	if len(node.Spec.Environment) != 0 {
		os.Environment = node.Spec.Environment
	}

	network := installer.Network{
		Interfaces: []installer.NetworkInterface{
			installer.NetworkInterface{
				Name:   node.Spec.Interface,
				HwAddr: node.Spec.MacAddress,
			},
		},
	}

	/*if len(node.Spec.Address) != 0 && len(node.Spec.Netmask) != 0 && len(node.Spec.Gateway) != 0 {
		network.IP = node.Spec.Address
		network.SubnetMask = node.Spec.Netmask
		network.Gateway = node.Spec.Gateway
	} */
	network.DefaultRoute = true
	network.Method = "dhcp"

	if len(node.Spec.NTPServers) != 0 {
		os.NTPServers = node.Spec.NTPServers
	}

	if len(node.Spec.DNSNameservers) != 0 {
		os.DNSNameservers = node.Spec.DNSNameservers
	}

	disk := "/dev/sda"
	if len(node.Spec.Disk) != 0 {
		disk = node.Spec.Disk
	}
	harvesterManagementNetwork := make(map[string]installer.Network)
	harvesterManagementNetwork["harvester-mgmt"] = network
	install := installer.Install{
		Networks:  harvesterManagementNetwork,
		Automatic: true,
		Mode:      "join",
		Device:    disk,
	}

	version, err := util.FindHarvesterVersion(c.Client)
	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "harvester version fetch error")
		return
	}

	if len(node.Spec.PXEIsoURL) != 0 {
		install.ISOURL = node.Spec.PXEIsoURL
	} else {
		install.ISOURL = fmt.Sprintf("https://releases.rancher.com/harvester/%s/harvester-%s-amd64.iso", version, version)
	}

	config := installer.HarvesterConfig{
		ServerURL: strings.Join(append([]string{"https"}, serverURLArr[1:len(serverURLArr)-1]...), ":") + ":8443",
		Token:     node.Spec.Token,
		OS:        os,
		Install:   install,
	}
	contentByte, err := yaml.Marshal(config)

	if err != nil {
		util.ReturnHTTPMessage(w, r, 500, "error", "error during config generation")
		return
	}

	util.ReturnHTTPRaw(w, r, string(contentByte))
}
