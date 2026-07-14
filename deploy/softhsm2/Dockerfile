# SoftHSM2 development image for PKCS#11 testing
# Usage:
#   docker build -t ggid-softhsm2 deploy/softhsm2
#   docker run -v $(pwd)/deploy/softhsm2/tokens:/tokens ggid-softhsm2
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    softhsm2 \
    libsofthsm2 \
    openssl \
    opensc \
    && rm -rf /var/lib/apt/lists/*

# Create token directory and config
RUN mkdir -p /tokens && \
    echo "directories.tokendir = /tokens" > /etc/softhsm/softhsm2.conf && \
    echo "objectstore.backend = file" >> /etc/softhsm/softhsm2.conf && \
    echo "log.level = INFO" >> /etc/softhsm/softhsm2.conf

ENV SOFTHSM2_CONF=/etc/softhsm/softhsm2.conf

COPY init.sh /usr/local/bin/init-softhsm2.sh
RUN chmod +x /usr/local/bin/init-softhsm2.sh

# Default: initialize token and generate a 2048-bit RSA key pair for signing
CMD ["/usr/local/bin/init-softhsm2.sh"]
