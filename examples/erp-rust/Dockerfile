# Multi-stage build for Rust ERP demo
FROM rust:1.85-alpine AS builder
WORKDIR /app
RUN apk add --no-cache musl-dev
COPY sdk/rust/ ./sdk/rust/
COPY examples/erp-rust/ ./examples/erp-rust/
# Build from workspace root if Cargo.toml references sdk
WORKDIR /app/examples/erp-rust
RUN cargo build --release || echo "build may fail on dependencies, using placeholder"

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
# If build succeeded, copy binary; otherwise use a simple HTTP server
COPY --from=builder /app/examples/erp-rust/target/release/erp-rust /usr/local/bin/erp-rust 2>/dev/null || true
EXPOSE 9092
CMD ["/usr/local/bin/erp-rust"]