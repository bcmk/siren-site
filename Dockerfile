FROM --platform=$BUILDPLATFORM node:24.4.1-alpine AS frontend
WORKDIR /app
COPY cmd/site/package.json cmd/site/yarn.lock ./
RUN yarn install --frozen-lockfile
COPY cmd/site ./
RUN yarn build

FROM golang:1.24.5-alpine AS gobuilder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
RUN apk add --no-cache bash git
COPY cmd/ cmd/
COPY sitelib/ sitelib/
COPY scripts/ scripts/
ARG SIREN_SITE_VERSION=devel
ENV CGO_ENABLED=0
ENV GOFLAGS=-trimpath
ENV LDFLAGS="-s -w"
RUN ./scripts/build-site "$SIREN_SITE_VERSION"

FROM alpine:3.22
WORKDIR /app/cmd/site
RUN apk add --no-cache ca-certificates
COPY --from=gobuilder /app/cmd/site/site ./site
COPY --from=gobuilder /app/cmd/site/icons/ ./icons/
COPY --from=gobuilder /app/cmd/site/pages/ ./pages/
COPY --from=frontend /app/static ./static
COPY --from=frontend /app/partial ./partial
ENTRYPOINT ["./site"]
CMD ["-c", "/app/config/config.yaml"]
