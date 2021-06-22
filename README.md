## Harvester tinkerbell operator
Simple operator to run on your Harvester cluster, to allow harvester to leverage tinkerbell to add more worker nodes to itself.

The operator uses the custom harvester installer in tinkerbell [boots](https://github.com/ibrokethecloud/boots/tree/harvester/installers/harvester).

The idea of this operator is to make it simpler to expand your harvester cluster using tinkerbell.

The operator can be installed via the harvester-tink-installer helm chart available in the `chart` directory

```bigquery
helm install harvester-tink-installer ./chart/harvester-tink-operator --create-namespace -n harvester-operator
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
  harvesterTinkOperator: gmehta3/harvester-tink-operator:dev
  boots: gmehta3/boots:harvester2
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
```


The operator will create the correct hardware object in tink and now the user can reboot said nodes to trigger the pxe based installation.

