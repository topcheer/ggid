FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 go build -o /identity ./services/identity/cmd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /identity /identity

EXPOSE 8081 50051
ENTRYPOINT ["/identity"]
