# syntax=docker/dockerfile:1
FROM --platform=linux/arm64 continuumio/miniconda3 AS conda
# Need to use conda because zfp doesn't work with pip
RUN conda install -c conda-forge zfpy
RUN conda install -c anaconda lz4
RUN conda install -c conda-forge keras
RUN conda install -c anaconda numpy
RUN conda install tensorflow

ARG fp=/root/

COPY ./inference_pod/inference_container/inference.py $fp
COPY ./io_util/pipes_util/ $fp
WORKDIR $fp

ENV PYTHONUNBUFFERED=1
ENTRYPOINT ["python", "inference.py"]