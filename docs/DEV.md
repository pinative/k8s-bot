# Development Mode

Running k8s-bot on your local machine

## Prerequisite

1. Golang 1.11+

2. An available Kubernetes cluster, e.g. minikube

3. kubectl must be installed and switch the context of your cluster to be .

## Running from sourcecode

1. Set `GO111MODULE=on`

2. `git clone` this repo

3. `cd k8s-bot` and rename the `.env-example` file to `.env`

4. `cmd && go build -o main`

5. `./main` to run it locally
