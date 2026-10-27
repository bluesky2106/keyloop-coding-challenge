# syntax=docker/dockerfile:1

FROM golang:1.26-alpine AS build
WORKDIR /src
# Dependencies first, so they are cached until go.mod or go.sum change.
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/keyloop-api ./cmd/api \
 && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/keyloop-seed ./cmd/seed

FROM alpine:3.22
RUN addgroup -S app && adduser -S -G app app
WORKDIR /app
# One image carries both programs: the API (default) and the one-shot demo data loader.
# Migrations are embedded in the binaries; the demo CSVs sit next to them.
COPY --from=build /out/keyloop-api /out/keyloop-seed ./
COPY seed ./seed
USER app
EXPOSE 8080
ENTRYPOINT ["/app/keyloop-api"]
