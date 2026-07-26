# syntax=docker/dockerfile:1.7

FROM node:24-alpine AS web
WORKDIR /src/web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ ./
RUN npm run build

FROM golang:1.24-alpine AS backend
WORKDIR /src
COPY go.mod ./
COPY assets/ ./assets/
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY --from=web /src/assets/dist ./assets/dist
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/leaseboard ./cmd/leaseboard

FROM alpine:3.22
RUN apk add --no-cache ca-certificates iproute2 tzdata \
    && addgroup -S leaseboard \
    && adduser -S -G leaseboard -h /nonexistent leaseboard
COPY --from=backend /out/leaseboard /usr/local/bin/leaseboard
USER leaseboard
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/leaseboard"]
