# syntax=docker/dockerfile:1
ARG fp=/root

FROM golang:latest as build_stage
ARG fp

COPY ./system_init_job/get_system_info_container/ ${fp}/src/system_init_job/get_system_info_container/
COPY ./system_init_job/get_node_bandwidths_container/ ${fp}/src/system_init_job/get_node_bandwidths_container/
COPY ./io_util/ ${fp}/src/io_util/

ENV GOPATH=$fp
# Initialize module in src directory (root of code files)
WORKDIR ${fp}/src
RUN go mod init github.com/Dat-Boi-Arjun/SEIFER
RUN go get k8s.io/api@latest
RUN go get k8s.io/client-go@latest
RUN go get k8s.io/apimachinery@latest
RUN go mod tidy

WORKDIR ${fp}/src/system_init_job/get_system_info_container/bin/
RUN env GOOS=linux GOARCH=arm go build -o ${fp}/main main.go

# Slimmed build stage only w/ executables
FROM --platform=linux/arm64 alpine:latest
ARG fp

WORKDIR $fp
COPY --from=build_stage ${fp}/main ./
COPY --from=build_stage ${fp}/src/system_init_job/get_system_info_container/pkg/dispatcher_configmap.sh ./
COPY --from=build_stage ${fp}/src/system_init_job/get_system_info_container/pkg/cluster_test.sh ./
COPY --from=build_stage ${fp}/src/system_init_job/get_system_info_container/pkg/configs/ ./configs/

# Install kubectl
RUN apk update && apk add curl && \
      curl -LO https://storage.googleapis.com/kubernetes-release/release/`curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt`/bin/linux/arm64/kubectl && \
      chmod +x ./kubectl && \
      mv ./kubectl /usr/local/bin/kubectl

ENTRYPOINT ["./main"]