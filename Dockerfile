FROM golang:1.18.3

RUN mkdir /updsrv-config

WORKDIR /updsrv

# RUN apt-get update
# RUN apt-get install -y htop mc tilde nano

COPY ./ .
RUN make build

EXPOSE 8080

ENTRYPOINT ["./updsrv", "-config-path", "/updsrv-config/config.toml"] 