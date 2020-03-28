# k8s-bot
A Kubernetes intelligent bot that enables you to manage Kubernetes resources automatically based on the configurations.

> NOTE: right now it only supports CREATE/UPDATE/DELETE Services and Ingresses (Nginx Ingress) for your Deployments. We 
will add more features over time, and any contributes and issues are welcome.

## To Deploy it onto your cluster

```bash 
wget https://github.com/pinative/k8s-bot/manifests/bot.yaml

kubectl apply -f bot.yaml
```

> **NOTE:** You have to replace the *PUBLIC_DNS_DOMAIN* to your dns in line 14 of bot.yaml before running the kubectl apply command.

## To Delete it from your cluster

`kubectl delete -f https://github.com/pinative/k8s-bot/manifests/bot.yaml`

## Configurations

* If you'd like to let k8s-bot automatically manage your Services and Ingresses, you have to add annotations
`"pigo.io/part-of": "k8s.bot"` and `"pigo.network/allow-internet-access":"true"` into your deployments respectively.

    > **NOTE:** You have to make sure the Nginx Ingress Controller is installed if you want to leveraging the ingress feature. 

* If you just need k8s-bot to manage your Services, then you just need to add an annotation `"pigo.io/part-of": "k8s.bot"`
into your deployments.

## DEV Mode

Please refer to [dev instruction](docs/DEV.md)

## Usage Examples

### 1. Manage Services and Ingresses

```bash
cat <<EOF >./html-edge.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: html-edge
  name: html-edge
  annotations: {"pigo.io/part-of": "k8s.bot", "pigo.network/allow-internet-access":"true"}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: html-edge
  template:
    metadata:
      labels:
        app: html-edge
    spec:
      containers:
      - image: pinative/html:edge
        name: html
        ports:
        - containerPort: 80
EOF

kubectl apply -f html-edge.yaml
```

### 2. Manage Services only

```bash
cat <<EOF >./html-edge.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  labels:
    app: html-latest
  name: html-latest
  annotations: {"pigo.io/part-of": "k8s.bot", "pigo.network/allow-internet-access":"false"}
spec:
  replicas: 1
  selector:
    matchLabels:
      app: html-latest
  template:
    metadata:
      labels:
        app: html-latest
    spec:
      containers:
      - image: pinative/html:latest
        name: html
        ports:
        - containerPort: 80
EOF

kubectl apply -f html-edge.yaml
```