FROM golang:1.25-alpine AS builder

WORKDIR /src
COPY .git/ ./.git/
COPY talk-libs/ ./talk-libs/
COPY mcp-playground/ ./mcp-playground/

WORKDIR /src/mcp-playground

ENV GOWORK=off

RUN VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev") && \
    go mod download && \
    CGO_ENABLED=0 go build \
        -ldflags "-X github.com/xvThomas/talk-backend/talk-libs/version.Version=${VERSION}" \
        -o /bin/mcp-playground \
        ./cmd

FROM alpine:3.21

RUN apk add --no-cache ca-certificates

COPY --from=builder /bin/mcp-playground /usr/local/bin/mcp-playground

USER appuser

EXPOSE 8080

ENTRYPOINT ["mcp-playground", "--transport", "http", "--addr", "0.0.0.0:8080"]

