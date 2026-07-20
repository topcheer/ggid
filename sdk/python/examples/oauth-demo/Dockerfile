FROM python:3.12-slim
WORKDIR /app
RUN pip install flask requests
COPY . .
ENV LISTEN_ADDR=:5000
EXPOSE 5000
CMD ["python", "app.py"]
