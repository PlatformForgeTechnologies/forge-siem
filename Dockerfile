FROM golang:1.23-alpine AS builder

ARG SERVICE

WORKDIR /src

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

RUN test -n "${SERVICE}"
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/service \
    ./cmd/${SERVICE}

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /out/service /usr/local/bin/forge-siem

ENTRYPOINT ["/usr/local/bin/forge-siem"]
