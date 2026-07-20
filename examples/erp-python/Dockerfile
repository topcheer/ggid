FROM python:3.12-slim

WORKDIR /app

# Copy project (SDK + demo)
COPY . /build

# Install SDK + extra runtime deps
RUN cd /build/sdk/python && pip install --no-cache-dir . && \
    pip install --no-cache-dir requests PyJWT cryptography fastapi uvicorn

# Copy demo app
RUN cp /build/examples/erp-python/main.py /app/main.py

WORKDIR /app
EXPOSE 9100

CMD ["python3", "main.py"]
