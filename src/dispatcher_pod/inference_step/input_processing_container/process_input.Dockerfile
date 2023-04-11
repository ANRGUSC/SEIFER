# syntax=docker/dockerfile:1
FROM --platform=linux/arm64 continuumio/miniconda3 AS conda
# Need to use conda because zfp doesn't work with pip
RUN conda install -c conda-forge zfpy
RUN conda install -c anaconda lz4
RUN conda install -c conda-forge keras
RUN conda install -c anaconda numpy
RUN conda install tensorflow
RUN conda install -c conda-forge keras
RUN conda install -c anaconda pillow

ARG fp=/root/

COPY ./dispatcher_pod/inference_step/input_processing_container/process_input.py $fp
COPY ./io_util/pipes_util/ $fp
COPY ./dispatcher_pod/inference_step/input_processing_container/elephant.jpg $fp
WORKDIR $fp

ENV PYTHONUNBUFFERED=1
ENTRYPOINT ["python", "process_input.py"]