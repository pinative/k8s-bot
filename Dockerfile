# Dockerfile References: https://docs.docker.com/engine/reference/builder/

# Start from the latest golang base image
FROM golang:latest as builder

LABEL maintainer="Zhantao Feng <feng@pinative.com>"

WORKDIR /app

COPY . .

# Download all dependencies and run go build.
#  Dependencies will be cached if the go.mod and go.sum files are not changed
RUN go mod download && cd cmd && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main .


######## Start a new stage from alpine #######
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/cmd/main .

CMD [ "./main" ]