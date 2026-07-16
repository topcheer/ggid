FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY deploy/operator/go.mod deploy/operator/go.sum ./
RUN go mod download
COPY deploy/operator/ .

RUN CGO_ENABLED=0 go build -o /operator ./cmd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates curl bash util-linux

# Install kubectl
ARG KUBECTL_VERSION=v1.31.2
RUN curl -fsSL "https://dl.k8s.io/release/${KUBECTL_VERSION}/bin/linux/$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')/kubectl" \
    -o /usr/local/bin/kubectl && chmod +x /usr/local/bin/kubectl

# Install helm
ARG HELM_VERSION=v3.16.3
RUN curl -fsSL "https://get.helm.sh/helm-${HELM_VERSION}-linux-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/').tar.gz" \
    | tar xz -C /tmp && mv /tmp/linux-*/helm /usr/local/bin/helm && chmod +x /usr/local/bin/helm

COPY --from=builder /operator /operator
COPY deploy/helm/ggid /charts/ggid

EXPOSE 8080 8081 9090
ENTRYPOINT ["/operator"]
