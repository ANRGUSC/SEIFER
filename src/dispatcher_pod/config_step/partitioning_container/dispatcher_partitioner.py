import pathlib

from partitioner import Partitioner
import networkx as nx
import json
import tensorflow as tf
from keras.applications import *
import os

cluster_storage_dir = "/nfs"
dispatcher_config_dir = "/dispatcher_config"
dispatcher_storage_dir = f"{cluster_storage_dir}/dispatcher_config"
model_save_dir = f"{cluster_storage_dir}/model_config"

# Init container will create node_info.json where each line represents a bandwidth between two nodes
# Can't put this in NFS b/c NFS hasn't been created yet
with open(f"{dispatcher_config_dir}/node_info.json", "r") as f:
    node_info = json.load(f)
    print(node_info)

communication_graph = nx.Graph()

for e in node_info["bandwidths"]:
    u = e["Start"]
    v = e["End"]
    bw = e["Bandwidth"]
    communication_graph.add_edge(u, v, weight=bw, name=f"{u}-{v}")

# We take the smallest RAM of all the nodes and make that the capacity, since the nodes should be homogeneous
# node_capacity = node_info["node_capacity"]
# Use mock node capacity for now since minikube nodes are much bigger
node_capacity = 64 * (1024 ** 2)
num_nodes = node_info["num_nodes"]
num_classes = 20

# Change this to later reference a model from a central repository
model = ResNet50(weights="imagenet")

partitioner = Partitioner(model)
candidate_part_pts, partitions, node_arrangement = partitioner.partitions_and_placement(num_nodes, num_classes,
                                                                                        node_capacity,
                                                                                        communication_graph)
print("Created model partitions")

print(node_arrangement)
# First node of partitions list is the dispatcher node, the rest are the compute nodes
for i in range(1, len(partitions)):
    part = partitions[i]
    # Model input
    if part[0] == 0:
        start_layer = candidate_part_pts[0]
    else:
        # construct_model() uses an exclusive start layer but inclusive end layer
        start_layer = candidate_part_pts[part[0] - 1]

    # Model output
    if part[1] == len(candidate_part_pts):
        end_layer = candidate_part_pts[-1]
    else:
        # End layer of partition in graph is exclusive, so need to subtract one from end layer index
        # to use with _construct_model(), which has inclusive end layer
        end_layer = candidate_part_pts[part[1] - 1]

    print(f"Partition {i}: ({start_layer}, {end_layer})")

    model = partitioner.construct_model(start_layer, end_layer, part_name=f"part_{i}")
    print("Partition constructed")

    node = node_arrangement[i]
    # The directory is the name of the node
    # The files in this directory is enough to create the inference pod
    directory = f"{model_save_dir}/partitions/{node}"

    # Convert to TF Lite model
    converter = tf.lite.TFLiteConverter.from_keras_model(model)
    # Quantize the model w/ float16 - big memory reduction w/ minimal accuracy loss
    converter.optimizations = [tf.lite.Optimize.DEFAULT]
    converter.target_spec.supported_types = [tf.float16]
    # converter.target_spec.supported_ops = [
    #     tf.lite.OpsSet.TFLITE_BUILTINS,  # enable TensorFlow Lite ops.
    #     tf.lite.OpsSet.SELECT_TF_OPS  # enable TensorFlow ops.
    # ]

    # Might throw error if it has layers not supported by TF Lite
    # For most production models it should be fine
    tflite_model = converter.convert()
    print("Converted partition #", str(i))

    # Writing to TF Lite file
    tflite_models_dir = pathlib.Path(directory)
    tflite_models_dir.mkdir(exist_ok=False, parents=True)
    tflite_model_file = tflite_models_dir / "model.tflite"
    tflite_model_file.write_bytes(tflite_model)
    print("Saved partition to %s" % str(tflite_model_file))

    # This file only has the next node (easiest way to make this info visible to the golang container)
    with open(f'{directory}/next_node.txt', 'x') as f:
        # n is the last node
        if i + 1 == len(node_arrangement):
            next_node = "dispatcher"
        else:
            next_node = node_arrangement[i + 1]
        f.write(next_node)

os.makedirs(dispatcher_storage_dir)
# File contains the node that the dispatcher pod needs to be scheduled on
with open(f"{dispatcher_storage_dir}/dispatcher_node.txt", 'x') as f:
    f.write(node_arrangement[0])

# File contains the next node of the dispatcher_pod
with open(f"{dispatcher_storage_dir}/dispatcher_next_node.txt", 'x') as f:
    f.write(node_arrangement[1])
