# syntax=docker/dockerfile:1
FROM --platform=linux/arm64 continuumio/miniconda3:latest AS conda
ARG fp=/root/

COPY ./dispatcher_pod/config_step/partitioning_container/ $fp

RUN conda install -c conda-forge keras
RUN conda install tensorflow
RUN conda install -c anaconda networkx

WORKDIR $fp
ENTRYPOINT ["python", "dispatcher_partitioner.py"]