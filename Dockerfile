FROM golang:1.21-alpine as build

COPY . /src

WORKDIR /src
RUN go mod download

WORKDIR /src/cmd/grash
RUN CGO_ENABLED=0 GOOS=linux go build -o /grash/

FROM ubuntu:latest

COPY --from=build /grash/grash /grash/
COPY --from=build /src/config/config.yaml /grash/
EXPOSE 8080

ENV LOG_PATH=/grash/logs/grash.log
ENV CONFIG_PATH=/grash/config.yaml
ENTRYPOINT ["/grash/grash"]