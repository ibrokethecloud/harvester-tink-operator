## Harvester tinkerbell operator
Simple operator to run on your Harvester cluster, to allow harvester to leverage tinkerbell to add more worker nodes to itself.

The operator uses the custom harvester installer in tinkerbell [boots](https://github.com/ibrokethecloud/boots/tree/harvester/installers/harvester).

The idea of this operator is to make it simpler to expand your harvester cluster using tinkerbell.

For now the operator assumes you already have tinkerbell running in your environment.

The quickly deploy the operator the following manifest can be used. The ConfigMap `tinkConfig` defines the tink endpoints:

```yaml

apiVersion: v1
kind: Namespace 
metadata:
  name: harvester-operator
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: harvester-tink-operator
  namespace: harvester-operator
  labels:
    operator: harvester-tink-operator
spec:
  selector:
    matchLabels:
      operator: harvester-tink-operator
  replicas: 1
  template:
    metadata:
      labels:
        operator: harvester-tink-operator
    spec:
      containers:
      - image: gmehta3/harvester-tink-operator:latest
        imagePullPolicy: Always
        name: manager
        ports:
        - containerPort: 30880
        resources:
          limits:
            cpu: 100m
            memory: 30Mi
          requests:
            cpu: 100m
            memory: 20Mi
      terminationGracePeriodSeconds: 10
      serviceAccountName: harvester-tink-operator
---
apiVersion: v1
kind: Service
metadata:
  name: harvester-tink-operator
  namespace: harvester-operator
  labels:
    operator: harvester-tink-operator
spec:
  type: NodePort
  ports:
  - port: 30880
    nodePort: 30880
    protocol: TCP
    targetPort: 30880
  selector:
    operator: harvester-tink-operator
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: harvester-tink-operator
  namespace: harvester-operator
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: harvester-tink-operator
subjects:
- kind: ServiceAccount
  name: harvester-tink-operator
  namespace: harvester-operator
roleRef:
  kind: ClusterRole
  name: cluster-admin
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: tinkconfig
  namespace: harvester-operator
data:
  CERT_URL: "http://172.16.132.186:42114/cert"
  GRPC_AUTH_URL: "172.16.132.186:42113"  

```

Once setup the user just needs to add additional nodes via the register.node CRD

```yaml
apiVersion: node.harvesterci.io/v1alpha1
kind: Register
metadata:
  name: node2
  namespace: harvester-operator
spec:
  # Add fields here
  macAddress: "0c:c4:7a:6b:80:d0"
  token: token
  interface: eth0
  harvesterIsoURL: http://172.16.132.186:8080/harvester-amd64.iso
  address: 172.16.128.11
  netmask: 255.255.248.0
  gateway: 172.16.128.1
```


The operator will create the correct hardware object in tink and now the user can reboot said nodes to trigger the pxe based installation.

*More detailed docs to follow* 