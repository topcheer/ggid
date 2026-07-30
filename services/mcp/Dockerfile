FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /mcp-server ./services/mcp/cmd/

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /mcp-server /usr/local/bin/mcp-server
RUN addgroup -S app && adduser -S app -G app
USER app
ENTRYPOINT ["mcp-server"]
