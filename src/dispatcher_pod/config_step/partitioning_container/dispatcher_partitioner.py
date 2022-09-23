# Use existing data in NFS for now
'''
import os
import pathlib

from partitioner import Partitioner
import networkx as nx
import json
import tensorflow as tf
from keras.applications import *


dispatcher_config_dir = "/dispatcher_config"
inference_pods_dir = "/nfs"
model_save_dir = f"{inference_pods_dir}/model_config"

# Init container will create node_info.json where each line represents a bandwidth between two nodes
with open(f"{dispatcher_config_dir}/node_info.json", "r") as f:
    node_info = json.load(f)
    print(node_info)

communication_graph = nx.Graph()

for e in node_info["bandwidths"]:
    u = e["Start"]
    v = e["End"]
    bw = e["Bandwidth"]
    inv_bandwidth = 1 / bw
    communication_graph.add_edge(u, v, weight=inv_bandwidth)

# We take the smallest RAM of all the nodes and make that the capacity, since the nodes should be homogeneous
node_capacities = [node_info["node_capacity"]] * node_info["num_nodes"]

# Change this to later reference a model from a central repository
model = ResNet50(weights="imagenet")

partitioner = Partitioner(model)
print("Partitioned model")
partitions = partitioner.create_model_partitions(node_capacities, communication_graph)
print("Created model partitions")
node_path = list(partitions)
for n in range(len(node_path)):
    node = node_path[n]
    # The directory is the name of the node
    # The files in this directory is enough to create the inference pod
    directory = f"{model_save_dir}/partitions/{node}"
    partition = partitions[node]
    # Convert to TF Lite model
    converter = tf.lite.TFLiteConverter.from_keras_model(partition)
    # Quantize the model w/ float16 - big memory reduction w/ minimal accuracy loss
    converter.optimizations = [tf.lite.Optimize.DEFAULT]
    converter.target_spec.supported_types = [tf.float16]
    # Might throw error if it has layers not supported by TF Lite
    # For most production models it should be fine
    tflite_model = converter.convert()
    print("Converted partition #", str(n+1))

    # Writing to TF Lite file
    tflite_models_dir = pathlib.Path(directory)
    tflite_models_dir.mkdir(exist_ok=True, parents=True)
    tflite_model_file = tflite_models_dir/"model.tflite"
    tflite_model_file.write_bytes(tflite_model)
    print("Saved partition to %s" % str(tflite_model_file))

    # For now this file only has the next node (easiest way to make this info visible to the golang container)
    # Can't use "x" because this file already exists for some reason
    with open(f'{directory}/next_node.txt', 'x') as f:
        # n is the last node
        if n+1 == len(node_path):
            # Need to include an if condition to check for this in k8s
            next_node = "dispatcher"
        else:
            next_node = node_path[n+1]
        f.write(next_node)

# File contains the next node of the dispatcher_pod
with open(f"{inference_pods_dir}/dispatcher_next_node.txt", 'x') as f:
    f.write(node_path[0])
'''


