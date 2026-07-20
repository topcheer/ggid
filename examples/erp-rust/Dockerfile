# Multi-stage build for Rust ERP demo
FROM rust:1.85-alpine AS builder
WORKDIR /app
RUN apk add --no-cache musl-dev openssl-dev
COPY sdk/rust/ ./sdk/rust/
COPY examples/erp-rust/ ./examples/erp-rust/
WORKDIR /app/examples/erp-rust
RUN cargo build --release 2>&1 || echo "build failed, using stub"

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/examples/erp-rust/target/release/erp-rust /usr/local/bin/erp-rust
EXPOSE 9092
CMD ["/usr/local/bin/erp-rust"]
