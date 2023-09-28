FROM golang:latest AS build-env
WORKDIR /src
ENV CGO_ENABLED=0
COPY go.* /src/
RUN go mod download
COPY . .
RUN go build -a -o app -ldflags="-s -w" -trimpath

FROM alpine:latest

RUN mkdir -p /app \
    && adduser -D user \
    && chown -R user:user /app

USER user
WORKDIR /app

COPY --from=build-env /src/app .

EXPOSE 8080

ENTRYPOINT [ "./app" ]
