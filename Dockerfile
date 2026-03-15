FROM golang:1.22-alpine AS builder

RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=1 go build -o dispatch -ldflags="-s -w" .

FROM alpine:3.20

RUN apk add --no-cache ca-certificates sqlite-libs
COPY --from=builder /build/dispatch /usr/local/bin/dispatch

WORKDIR /etc/dispatch
VOLUME ["/etc/dispatch", "/var/lib/dispatch"]

EXPOSE 8080

ENTRYPOINT ["dispatch"]
CMD ["serve", "/etc/dispatch/dispatch.yaml"]
