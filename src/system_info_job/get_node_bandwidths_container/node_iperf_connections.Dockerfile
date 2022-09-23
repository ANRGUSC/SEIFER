# syntax=docker/dockerfile:1
ARG fp=/root

FROM golang:latest as build_stage
ARG fp

COPY ./system_info_job/get_node_bandwidths_container/ ${fp}/src/system_info_job/get_node_bandwidths_container/
COPY ./io_util/ ${fp}/src/io_util/

ENV GOPATH=$fp
# Initialize module in src directory (root of code files)
WORKDIR ${fp}/src
RUN go mod init github.com/Dat-Boi-Arjun/DEFER
RUN go get github.com/pbnjay/memory
RUN go mod tidy
WORKDIR ${fp}/src/system_info_job/get_node_bandwidths_container/bin/
RUN env GOOS=linux GOARCH=arm go build -o ${fp}/main main.go

FROM ubuntu:latest as build-iperf
ARG fp

RUN apt update -y && apt upgrade -y
RUN apt-get update
RUN apt install wget -y
RUN apt install git -y
RUN apt install build-essential -y
RUN git clone https://github.com/esnet/iperf.git
WORKDIR iperf
RUN git checkout 3.11
RUN ./bootstrap.sh; ./configure --prefix=/usr; make; make install
#RUN iperf3 --help
RUN apt-get autoclean -y
RUN apt-get clean -y
RUN apt install iputils-ping -y
RUN apt install iproute2 -y
RUN apt-get install telnet

WORKDIR $fp
COPY --from=build_stage ${fp}/main ./

WORKDIR $fp
ENTRYPOINT ["./main"]