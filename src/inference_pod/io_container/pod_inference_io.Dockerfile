# syntax=docker/dockerfile:1
ARG fp=/root

FROM golang:latest as build_stage
ARG fp

COPY ./inference_pod/io_container/ ${fp}/src/inference_pod/io_container/
COPY ./io_util/ ${fp}/src/io_util/

ENV GOPATH=$fp
# Initialize module in src directory (root of code files)
WORKDIR ${fp}/src
RUN go mod init github.com/Dat-Boi-Arjun/DEFER

WORKDIR ${fp}/src/inference_pod/io_container/bin/
RUN env GOOS=linux GOARCH=arm go build -o ${fp}/main main.go

# Slimmed build stage only w/ executable
FROM alpine:latest
ARG fp

WORKDIR $fp
COPY --from=build_stage ${fp}/main ./

ENTRYPOINT ["./main"]