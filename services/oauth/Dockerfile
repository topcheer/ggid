FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 go build -o /oauth-server ./services/oauth/cmd/

FROM alpine:3.21
RUN apk add --no-cache ca-certificates tzdata
COPY --from=builder /oauth-server /usr/local/bin/oauth-server

EXPOSE 9005
ENTRYPOINT ["oauth-server"]
