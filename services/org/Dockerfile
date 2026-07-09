FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN CGO_ENABLED=0 go build -o /org ./services/org/cmd

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /org /org

EXPOSE 8071 9071
ENTRYPOINT ["/org"]
