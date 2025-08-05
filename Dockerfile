FROM node:24.4.1-alpine AS frontend
WORKDIR /app
COPY cmd/site/package.json cmd/site/yarn.lock ./
RUN yarn install --frozen-lockfile

FROM golang:1.24.5-alpine
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
RUN apk add bash git
COPY cmd/ cmd/
COPY sitelib/ sitelib/
COPY scripts/ scripts/

ARG SIREN_SITE_VERSION=devel
RUN ./scripts/build-all "$SIREN_SITE_VERSION"
COPY --from=frontend /app/node_modules cmd/site/node_modules

WORKDIR /app/cmd/site
CMD ["./site", "/app/config/config.yaml"]
