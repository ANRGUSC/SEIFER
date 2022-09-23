# syntax=docker/dockerfile:1
FROM --platform=linux/arm64 continuumio/miniconda3:latest AS conda
ARG fp=/root/

COPY ./dispatcher_pod/config_step/partitioning_container/ $fp

RUN conda install -c conda-forge keras
RUN conda install tensorflow
RUN conda install -c anaconda networkx
RUN conda install -c conda-forge bidict

RUN wget "http://www.math.uwaterloo.ca/tsp/concorde/downloads/codes/linux24/concorde.gz"
RUN gzip -d concorde.gz
RUN chmod +x concorde
RUN mv concorde /usr/local/bin
RUN pip install tsplib95

WORKDIR $fp
ENTRYPOINT ["python", "dispatcher_partitioner.py"]