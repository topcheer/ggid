FROM golang:1.23-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /demo ./sdk/go/examples/demo/

FROM alpine:3.20
COPY --from=builder /demo /demo
EXPOSE 9090
CMD ["/demo"]
