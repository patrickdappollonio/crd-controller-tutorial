FROM alpine as certs
RUN apk add --no-cache ca-certificates && update-ca-certificates

FROM golang:1.20 as builder
WORKDIR /app
ADD . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -extldflags='-static'" -o /bin/controller

FROM scratch
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=builder /bin/controller /controller
EXPOSE 8080
ENTRYPOINT ["/controller"]

