FROM node:18-alpine
WORKDIR /app
ENV NODE_ENV=development
COPY package.json ./
RUN npm install
COPY . .
EXPOSE 3300
CMD ["npx", "next", "dev", "-p", "3300", "-H", "0.0.0.0"]