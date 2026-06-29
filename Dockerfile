FROM golang:1.25-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY . .
RUN CGO_ENABLED=0 go build -o /out/convertthis ./cmd/server

FROM alpine:3.20
RUN apk add --no-cache ffmpeg
COPY --from=build /out/convertthis /usr/local/bin/convertthis
ENV ADDR=:8080
EXPOSE 8080
ENTRYPOINT ["convertthis"]
