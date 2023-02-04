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
    communication_graph.add_edge(u, v, weight=bw)

# We take the smallest RAM of all the nodes and make that the capacity, since the nodes should be homogeneous
node_capacities = [node_info["node_capacity"]] * node_info["num_nodes"]

# Change this to later reference a model from a central repository
model = ResNet50(weights="imagenet")

partitioner = Partitioner(model)
print("Partitioned model")
node_arrangement, partitions = partitioner.construct_models
print("Created model partitions")
for n in range(len(node_arrangement)):
    node = node_arrangement[n]
    # The directory is the name of the node
    # The files in this directory is enough to create the inference pod
    directory = f"{model_save_dir}/partitions/{node}"
    partition = partitions[n]
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
    with open(f'{directory}/next_node.txt', 'x') as f:
        # n is the last node
        if n+1 == len(node_arrangement):
            next_node = "dispatcher"
        else:
            next_node = node_arrangement[n+1]
        f.write(next_node)

# File contains the next node of the dispatcher_pod
with open(f"{inference_pods_dir}/dispatcher_next_node.txt", 'x') as f:
    # The dispatcher is scheduled to any random node, so we have to hope that it won't have a high latency
    # connection to the first compute node
    f.write(node_arrangement[0])



