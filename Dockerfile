
FROM golang:1.15.0-alpine3.12 as builder

COPY . /build/
WORKDIR /build
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o broxy2socks .

FROM scratch
COPY --from=builder /build/broxy2socks /broxy2socks
ENTRYPOINT [ "/broxy2socks" ]
