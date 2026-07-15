FROM node:22-alpine AS sdk-builder
WORKDIR /sdk
COPY sdk/react/package.json sdk/react/tsconfig.json ./
COPY sdk/react/src ./src
RUN npm install --production=false && npm run build

FROM node:22-alpine AS builder
WORKDIR /app
COPY console/package.json ./
RUN npm install --no-package-lock

# Copy compiled SDK into node_modules
COPY --from=sdk-builder /sdk/package.json /tmp/sdk-pkg.json
COPY --from=sdk-builder /sdk/dist /tmp/sdk-dist
RUN mkdir -p node_modules/@ggid/sdk-react && \
    cp /tmp/sdk-pkg.json node_modules/@ggid/sdk-react/package.json && \
    cp -r /tmp/sdk-dist node_modules/@ggid/sdk-react/dist

# Copy console source (skip lock file to avoid file: dependency issues)
COPY console/ ./
ENV NEXT_TELEMETRY_DISABLED=1
# Empty = relative URLs (proxied via next.config rewrites to gateway)
ARG NEXT_PUBLIC_API_URL=
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
ARG NEXT_PUBLIC_TENANT_ID=00000000-0000-0000-0000-000000000001
ENV NEXT_PUBLIC_TENANT_ID=$NEXT_PUBLIC_TENANT_ID
# GATEWAY_URL must be set at build time for Next.js standalone rewrites
ARG GATEWAY_URL=http://ggid-gateway:8080
ENV GATEWAY_URL=$GATEWAY_URL
RUN npm run build

FROM node:22-alpine
WORKDIR /app
ENV NEXT_TELEMETRY_DISABLED=1
ENV NODE_ENV=production
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static
COPY --from=builder /app/public ./public
EXPOSE 3000
ENV HOSTNAME=0.0.0.0
CMD ["node", "server.js"]