FROM golang:1.18.3

RUN mkdir /logsrv-config

WORKDIR /logsrv

RUN apt-get update
RUN apt-get install -y htop mc tilde nano

COPY ./ .
RUN go install github.com/google/wire/cmd/wire@latest
RUN make rebuild

EXPOSE 8080

ENTRYPOINT ["./logsrv", "-config-path", "/logsrv-config/config.toml"] 