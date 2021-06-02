package tink

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	nodev1alpha1 "github.com/ibrokethecloud/harvester-tink-operator/api/v1alpha1"

	"github.com/tinkerbell/tink/protos/hardware"

	"github.com/pkg/errors"
	hw "github.com/tinkerbell/tink/client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func NewClient(apiClient client.Client) (fullClient *hw.FullClient, err error) {
	var certURL, grpcAuth string
	cm := &corev1.ConfigMap{}
	err = apiClient.Get(context.Background(), types.NamespacedName{Name: nodev1alpha1.ConfigMapName, Namespace: nodev1alpha1.ConfigMapNamespace}, cm)
	if err != nil {
		return nil, errors.Wrap(err, "error during configMap get")
	}

	certURL, ok := cm.Data["CERT_URL"]
	if !ok {
		return nil, fmt.Errorf("cert_url not found in configmap tinkConfig")
	}

	grpcAuth, ok = cm.Data["GRPC_AUTH_URL"]
	if !ok {
		return nil, fmt.Errorf("grpc_auth_url not found in configmap tinkConfig")
	}

	connOpts := &hw.ConnOptions{CertURL: certURL, GRPCAuthority: grpcAuth}

	grpcConn, err := hw.NewClientConn(connOpts)
	if err != nil {
		return nil, errors.Wrap(err, "error creating grpc clients")
	}

	fullClient = hw.NewFullClient(grpcConn)

	return fullClient, err
}

func GenerateHWRequest(regoReq *nodev1alpha1.Register, serverURL string) (hw *hardware.Hardware, err error) {

	networkInterfaces := &hardware.Hardware_Network_Interface{
		Netboot: &hardware.Hardware_Netboot{
			AllowPxe: true,
		},
	}

	// Specify non default location to load ISO's
	if len(regoReq.Spec.IsoURL) != 0 {
		networkInterfaces.Netboot.Osie = &hardware.Hardware_Netboot_Osie{BaseUrl: regoReq.Spec.IsoURL}
	}

	ip := &hardware.Hardware_DHCP_IP{}

	dhcpRequest := &hardware.Hardware_DHCP{
		Mac:       regoReq.Spec.MacAddress,
		IfaceName: regoReq.Spec.Interface,
		Ip:        ip,
	}

	if len(regoReq.Spec.Address) != 0 && len(regoReq.Spec.Gateway) != 0 && len(regoReq.Spec.Netmask) != 0 {
		ip.Address = regoReq.Spec.Address
		ip.Gateway = regoReq.Spec.Gateway
		ip.Netmask = regoReq.Spec.Netmask

	}

	// update dhcp request
	networkInterfaces.Dhcp = dhcpRequest

	url, err := url.Parse(serverURL)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing server url")
	}

	urlArr := strings.Split(url.Host, ":")

	// url for fetching config data for new node //
	configUrl := fmt.Sprintf("http://%s:%s/config/%s", urlArr[0], nodev1alpha1.DefaultConfigURLPort, regoReq.Status.UUID)

	userData := fmt.Sprintf("harvester.install.config_url=%s", configUrl)
	m := nodev1alpha1.MetaData{
		Facility: nodev1alpha1.Facility{
			FacilityCode: "onprem",
		},
		Instance: nodev1alpha1.Instance{
			UserData: userData,
			OperatingSystem: nodev1alpha1.OperatingSystem{
				Slug: "harvester_0_2_0",
			},
		},
	}

	mByte, err := json.Marshal(m)
	if err != nil {
		return hw, errors.Wrap(err, "error marshalling metadata")
	}

	hw = &hardware.Hardware{
		Id: regoReq.Status.UUID,
		Network: &hardware.Hardware_Network{
			Interfaces: []*hardware.Hardware_Network_Interface{
				networkInterfaces,
			},
		},
		Metadata: string(mByte),
	}

	return hw, nil
}
