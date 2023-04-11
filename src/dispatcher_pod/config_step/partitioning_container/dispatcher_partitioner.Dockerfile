# syntax=docker/dockerfile:1
FROM --platform=linux/arm64 continuumio/miniconda3:latest AS conda
ARG fp=/root/

RUN conda install -c conda-forge keras
RUN conda install tensorflow
RUN conda install -c anaconda networkx

COPY ./dispatcher_pod/config_step/partitioning_container/ $fp

WORKDIR $fp
ENV PYTHONUNBUFFERED=1
ENTRYPOINT ["python", "dispatcher_partitioner.py"]