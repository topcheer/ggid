# ggid-cli build stage
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bin/ggid ./services/ggid-cli/cmd/

# Final image
FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /bin/ggid /usr/local/bin/ggid
ENTRYPOINT ["ggid"]
