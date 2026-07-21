FROM node:18-alpine
WORKDIR /app
ENV NODE_ENV=development
ENV NEXT_TELEMETRY_DISABLED=1
COPY examples/erp-react/package.json ./
RUN npm install
COPY examples/erp-react/ ./
EXPOSE 3300
CMD ["npx", "next", "dev", "-p", "3300", "-H", "0.0.0.0"]
