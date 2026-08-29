# syntax=docker/dockerfile:1

FROM node:26-bookworm-slim AS frontend
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.26.7-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
COPY --from=frontend /src/web/dist ./internal/webui/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/leakboard ./cmd/leakboard

FROM debian:bookworm-slim AS gitleaks
ARG GITLEAKS_VERSION=8.30.1
ARG TARGETARCH
RUN apt-get update && apt-get install -y --no-install-recommends curl ca-certificates && rm -rf /var/lib/apt/lists/*
RUN set -eux; \
	case "$TARGETARCH" in \
		amd64) GL_ARCH=x64 ;; \
		arm64) GL_ARCH=arm64 ;; \
		*) echo "unsupported arch: $TARGETARCH" >&2; exit 1 ;; \
	esac; \
	curl -fsSL "https://github.com/gitleaks/gitleaks/releases/download/v${GITLEAKS_VERSION}/gitleaks_${GITLEAKS_VERSION}_linux_${GL_ARCH}.tar.gz" \
	| tar -xz -C /usr/local/bin gitleaks

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends git ca-certificates && rm -rf /var/lib/apt/lists/*
COPY --from=gitleaks /usr/local/bin/gitleaks /usr/local/bin/gitleaks
COPY --from=build /out/leakboard /usr/local/bin/leakboard

RUN useradd --create-home --shell /usr/sbin/nologin leakboard \
	&& mkdir -p /home/leakboard/data/repos \
	&& chown -R leakboard:leakboard /home/leakboard/data
USER leakboard
WORKDIR /home/leakboard
ENV LEAKBOARD_ADDR=:8080
ENV LEAKBOARD_WORKDIR=/home/leakboard/data/repos
EXPOSE 8080

ENTRYPOINT ["leakboard"]
