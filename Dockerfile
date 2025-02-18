FROM golang:1.24 as builder

ENV CGO_ENABLED=0

WORKDIR /app
COPY . .
RUN go build -o bin/xbot ./cmd/main.go


FROM ghcr.io/orvice/go-runtime:master
WORKDIR /app
COPY --from=builder /app/bin/xbot /app/bin/xbot
ENTRYPOINT ["/app/bin/xbot"]