FROM rust:1.88-alpine AS builder
WORKDIR /app
RUN apk add --no-cache musl-dev openssl-dev openssl-libs-static
COPY sdk/rust/ ./sdk/rust/
COPY examples/erp-rust/ ./examples/erp-rust/
WORKDIR /app/examples/erp-rust
ENV OPENSSL_STATIC=1 OPENSSL_LIB_DIR=/usr/lib OPENSSL_INCLUDE_DIR=/usr/include
RUN cargo build --release

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/examples/erp-rust/target/release/erp-rust /usr/local/bin/erp-rust
EXPOSE 9092
CMD ["/usr/local/bin/erp-rust"]
