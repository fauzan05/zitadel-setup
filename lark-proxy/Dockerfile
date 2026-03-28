# syntax=docker/dockerfile:1

FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod ./
RUN go mod download
COPY main.go ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/lark-proxy .

FROM alpine:3.19
RUN apk add --no-cache ca-certificates
COPY --from=build /out/lark-proxy /usr/local/bin/lark-proxy
EXPOSE 4000
ENV PORT=4000
ENTRYPOINT ["/usr/local/bin/lark-proxy"]
