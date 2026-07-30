FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 go build -o /policy ./services/policy/cmd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates && adduser -D -u 1001 appuser
COPY --from=builder /policy /policy
USER appuser

EXPOSE 8070 9070
ENTRYPOINT ["/policy"]
