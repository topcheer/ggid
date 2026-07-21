FROM node:18-alpine AS builder
WORKDIR /app
COPY examples/erp-node/package.json ./
RUN npm install
COPY examples/erp-node/src ./src

FROM node:18-alpine AS runner
WORKDIR /app
COPY examples/erp-node/package.json ./
RUN npm install --production
COPY --from=builder /app/src ./src
COPY --from=builder /app/node_modules ./node_modules
# SDK at /sdk/node/ to match demo import path ../../../sdk/node/src/client
COPY sdk/node/package.json /sdk/node/package.json
COPY sdk/node/src /sdk/node/src
RUN cd /sdk/node && npm install --production 2>/dev/null || true
ENV NODE_ENV=production
EXPOSE 3200
CMD ["npx", "tsx", "src/index.ts"]
