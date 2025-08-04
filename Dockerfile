FROM node:24.4.1-alpine AS frontend
WORKDIR /app
COPY cmd/site/package.json cmd/site/yarn.lock ./
RUN yarn install --frozen-lockfile

FROM golang:1.24.5-alpine
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN apk add bash git
RUN ./scripts/build-all
COPY --from=frontend /app/node_modules cmd/site/node_modules
RUN cp site cmd/site

WORKDIR /app/cmd/site
CMD ["./site", "/app/config/config.yaml"]
