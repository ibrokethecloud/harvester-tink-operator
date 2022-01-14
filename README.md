## Harvester tinkerbell operator
Simple operator to run on your Harvester cluster, to allow harvester to leverage tinkerbell to add more worker nodes to itself.

The operator uses the custom harvester installer in tinkerbell [boots](https://github.com/ibrokethecloud/boots/tree/harvester/installers/harvester).

The idea of this operator is to make it simpler to expand your harvester cluster using tinkerbell.

The operator can be installed via the harvester-tink-installer helm chart available in the `chart` directory

```bigquery
helm install harvester-tink-installer ./chart/harvester-tink-installer --create-namespace -n harvester-operator
```

By default the helm chart will deploy a minimal tink setup on the Harvester node itself, which will then subsequently be used for serving pxe boot of additional nodes.

We use a custom image of boots which supports harvester pxe boot logic.

```yaml
## Does tink need to be installed on cluster
## Default is "true"
tinkInstall: true

# If tinkInstall is set to false, then we can specify external tinkerbell setup
tinkCertURL: "remote_tink_cert_url"
tinkGrpcAuthURL: "remote_tink_grpc_auth_url"

images:
  harvesterTinkOperator: gmehta3/harvester-tink-operator:harvester3
  boots: gmehta3/boots:harvester3
  tinkCli: quay.io/tinkerbell/tink-cli:sha-1b178dae
  tinkServer: quay.io/tinkerbell/tink:sha-1b178dae
  tinkServerInit: busybox:latest
  db: docker.io/postgres:10-alpine
```

Once setup the user just needs to add additional nodes via the register.node CRD

```yaml
apiVersion: node.harvesterci.io/v1alpha1
kind: Register
metadata:
  name: node2
spec:
  # Add fields here
  macAddress: "0c:c4:7a:6b:80:d0"
  token: token
  interface: eth0
  address: 172.16.128.11
  netmask: 255.255.248.0
  gateway: 172.16.128.1
  pxeIsoURL: http://172.16.135.50:8080/v1.0.0/harvester-v1.0.0-amd64.iso #Optional. If not specified operator will use the appropriate iso version eg.. https://releases.rancher.com/harvester/v0.3.0/harvester-v0.3.0-amd64.iso
  imageURL: http://172.16.135.50:8080 #Optional argument to specify where to find kernel, initrd  and rootfs images. 
  slug: "harvester_1_0_0" #Version of Harvester to install
  kernelBootArguments: "ip=enp0s20f0:dhcp" #Additional kernel arguments. Currently ip arguments are needed to work around issue: https://github.com/harvester/harvester/issues/1363
```


The operator will create the correct hardware object in tink and now the user can reboot said nodes to trigger the pxe based installation.

**NOTE for airgapped environments**

If imageURL is specified, then please ensure that the correct version folder with artifact names exists.

For example if imageURL is http://172.16.135.50:8080, then harvester boots image will look for artifacts at the following location

```
initrd: http://172.16.135.50:8080/v1.0.0/harvester-v1.0.0-initrd-amd64
kernel: http://172.16.135.50:8080/v1.0.0/harvester-v1.0.0-vmlinuz-amd64
rootfs: http://172.16.135.50:8080/v1.0.0/harvester-v1.0.0-rootfs-amd64.squashfs
```

