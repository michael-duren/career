FROM node:22-bookworm-slim AS frontend
WORKDIR /build
COPY package.json package-lock.json ./
RUN npm ci
COPY astro.config.mjs tsconfig.json tailwind.config.mjs ./
COPY src ./src
COPY public ./public
RUN npm run build

FROM golang:1.27.0-bookworm AS backend
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY internal ./internal
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /career ./cmd/api

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=backend /career /app/career
COPY --from=frontend /build/dist /app/dist
USER 65532:65532
ENV APP_ENV=production LISTEN_ADDR=:8080 STATIC_DIR=/app/dist
EXPOSE 8080
ENTRYPOINT ["/app/career"]
CMD ["serve"]

