# ============================================================================
# BUILDER STAGE
# ============================================================================
FROM golang:1.25-alpine AS builder

ARG SERVICE=gateway

WORKDIR /build

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=linux go build \
        -ldflags="-s -w" \
        -o /dist/app ./cmd/${SERVICE}

# ============================================================================
# RUN STAGE
# ============================================================================
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata wget

WORKDIR /app

COPY --from=builder /dist/app /app/server

ENTRYPOINT ["/app/server"]
