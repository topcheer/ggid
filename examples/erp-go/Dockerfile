FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN cd examples/erp-go && CGO_ENABLED=0 go build -o /erp-go .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /erp-go /usr/local/bin/erp-go
ENV ERP_LISTEN=:9090
EXPOSE 9090
CMD ["erp-go"]