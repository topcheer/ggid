FROM node:18-alpine AS builder
WORKDIR /app
COPY package.json ./
RUN npm install
COPY . .
RUN npx tsc --outDir dist --module commonjs --moduleResolution node --target es2020 --esModuleInterop --skipLibCheck src/index.ts 2>/dev/null || true

FROM node:18-alpine AS runner
WORKDIR /app
COPY package.json ./
RUN npm install --production
COPY --from=builder /app/src ./src
COPY --from=builder /app/node_modules ./node_modules
ENV NODE_ENV=production
EXPOSE 3200
CMD ["npx", "tsx", "src/index.ts"]
