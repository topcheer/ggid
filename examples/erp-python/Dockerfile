FROM python:3.12-slim

WORKDIR /app

# Copy Python SDK
COPY sdk/python/ggid ./ggid
COPY sdk/python/requirements.txt ./requirements.txt

# Install dependencies
RUN pip install --no-cache-dir -r requirements.txt

# Copy demo
COPY examples/erp-python/main.py ./main.py

EXPOSE 9100

CMD ["python3", "main.py"]
