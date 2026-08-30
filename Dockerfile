# syntax=docker/dockerfile:1

# Build the CLI as a fully static binary so it can run on a minimal base image.
FROM golang:1.25-alpine AS build

ARG VERSION=dev
ARG COMMIT=none
ARG DATE=unknown
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src

# Download modules first so this layer is cached until go.mod/go.sum change.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

ENV CGO_ENABLED=0
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath \
    -ldflags="-s -w \
      -X github.com/arthurlch/goryu/internal/cli.version=${VERSION} \
      -X github.com/arthurlch/goryu/internal/cli.commit=${COMMIT} \
      -X github.com/arthurlch/goryu/internal/cli.date=${DATE}" \
    -o /out/goryu ./cmd/goryu

FROM gcr.io/distroless/static:nonroot

COPY --from=build /out/goryu /usr/local/bin/goryu

USER nonroot:nonroot
ENTRYPOINT ["goryu"]
