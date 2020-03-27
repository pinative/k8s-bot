# k8s-bot
A Kubernetes intelligent bot that enables you to manage Kubernetes resources automatically based on the configurations.

> NOTE: right now it only supports CREATE/UPDATE/DELETE Services and Ingresses (Nginx Ingress) for your Deployments. We 
will add more features over time, but welcome any contributes and issues if you'd like to see additional features you
can create a 

# Development Mode

## Prerequisite

1. Golang 1.11+

2. kubectl must be installed and the context of your cluster should be configured on your local if you are running this 
in development mode.

## Running Steps

1. Set `GO111MODULE=on`

2. `git clone` this repo

3. `cd k8s-bot` and rename the `.env-example` file to `.env`

4. `cmd && go build`

5. `./cmd` to run this locally


# Production Mode

## To Deploy onto your cluster

`kubectl apply -f https://github.com/pinative/k8s-bot/manifests/bot.yaml`
