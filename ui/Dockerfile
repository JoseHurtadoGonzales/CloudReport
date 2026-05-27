# ── Build stage ────────────────────────────────────────────────────────────
FROM node:20-alpine AS build
WORKDIR /app

# Install deps first (better layer caching).
COPY package.json package-lock.json* ./
RUN npm ci

# Build the Nuxt app (Nitro node-server preset → .output/server/index.mjs).
COPY . .
# NUXT_PUBLIC_API_BASE is baked at build time for the client bundle; it can
# still be overridden at runtime for the server side via the same env var.
ARG NUXT_PUBLIC_API_BASE=http://localhost:5488
ENV NUXT_PUBLIC_API_BASE=$NUXT_PUBLIC_API_BASE
RUN npm run build

# ── Runtime stage ──────────────────────────────────────────────────────────
FROM node:20-alpine AS runtime
WORKDIR /app
ENV NODE_ENV=production
ENV NITRO_HOST=0.0.0.0
ENV NITRO_PORT=3030

# Only the build output is needed at runtime — no node_modules, no source.
COPY --from=build /app/.output ./.output

EXPOSE 3030
CMD ["node", ".output/server/index.mjs"]
