FROM node:20-alpine
WORKDIR /app
RUN npm install express tsx typescript @types/express @types/node
COPY . .
ENV LISTEN_ADDR=:3000
EXPOSE 3000
CMD ["npx", "tsx", "index.ts"]
