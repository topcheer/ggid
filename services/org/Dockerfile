FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 go build -o /org ./services/org/cmd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /org /org
RUN addgroup -S app && adduser -S app -G app
USER app

EXPOSE 8071 9071
ENTRYPOINT ["/org"]
