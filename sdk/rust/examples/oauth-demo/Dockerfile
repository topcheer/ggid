FROM rust:1.79-slim
WORKDIR /app
COPY Cargo.toml .
COPY src/ src/
RUN cargo build --release
ENV PORT=8080
EXPOSE 8080
CMD ["./target/release/oauth-demo"]
