import networkx as nx
from tensorflow import keras
import tensorflow as tf
from typing import List

from graph_utils import GraphUtils


class Partitioner:
    def __init__(self, model: keras.Model):
        self.model = model
        self.Stack = []
        self.visited = {}
        # The "depth"/level that a certain layer is at
        self.layer_level = {}
        # The layers at a certain depth/level, where the index of the array is the level
        self.levels = []

        self.graph_utils = GraphUtils()

    def get_previous(self, layer_name):
        inbound = self.model.get_layer(layer_name).inbound_nodes[0].inbound_layers
        if type(inbound) != list:
            inbound = [inbound]
        return [layer.name for layer in inbound]

    def get_next(self, layer_name):
        outbound = self.model.get_layer(layer_name).outbound_nodes
        return [node.outbound_layer.name for node in outbound]

    # Constructs model using shape of start layer as the input (doesn't include start layer in the model)
    def construct_model(self, start, end, part_name="part_begin"):
        inpt = keras.Input(tensor=self.model.get_layer(start).output, name=part_name)
        outpt = self.model.get_layer(end).output
        part = keras.Model(inputs=inpt, outputs=outpt)
        return part

    def partitions_and_placement(self, num_nodes: int, num_classes: int, node_capacity: int, G_c: nx.Graph):
        part_pts = self.find_partitions()
        transfers = self.find_partition_transfer_size(part_pts)
        partition_mems = self.find_partition_memory(part_pts)
        partitions, node_arrangement = self.graph_utils.partition_and_place(num_nodes, node_capacity, G_c, num_classes,
                                                                            part_pts, transfers, partition_mems)

        return part_pts, partitions, node_arrangement

    # A recursive function used by longest_path. See below
    # link for details
    # https:#www.geeksforgeeks.org/topological-sorting/
    def topological_sort_util(self, v: str):
        self.visited[v] = True

        # Recur for all the vertices adjacent to this vertex
        # list<AdjListNode>::iterator i
        for i in self.get_next(v):
            if not self.visited[i]:
                self.topological_sort_util(i)

        # Push current vertex to stack which stores topological
        # sort
        self.Stack.append(v)

    # The function to find longest distances from a given vertex.
    # It uses recursive topologicalSortUtil() to get topological
    # sorting.
    def longest_path(self, s: str) -> List[List[str]]:
        for l in self.model.layers:
            self.visited[l.name] = False
            self.layer_level[l.name] = -1  # Equal to -infty

        # Call the recursive helper function to store Topological
        # Sort starting from all vertices one by one
        for l in self.model.layers:
            if not self.visited[l.name]:
                self.topological_sort_util(l.name)

        # Initialize distances to all vertices as infinite and
        # distance to source as 0
        self.layer_level[s] = 0

        # Process vertices in topological order
        while len(self.Stack) > 0:

            # Get the next vertex from topological order
            u = self.Stack.pop()

            # Update distances of all adjacent vertices
            # list<AdjListNode>::iterator i
            if self.layer_level[u] != -1:
                for i in self.get_next(u):
                    if self.layer_level[i] < self.layer_level[u] + 1:
                        self.layer_level[i] = self.layer_level[u] + 1  # Each edge weighted 1

        # Create array of calculated longest distances to layer
        layers_at_level = [[]] * len(self.layer_level)
        for l in self.model.layers:
            if len(layers_at_level[self.layer_level[l.name]]) == 0:
                layers_at_level[self.layer_level[l.name]] = []

            layers_at_level[self.layer_level[l.name]].append(l.name)

        return layers_at_level

    def find_singletons(self):
        # Model only has 1 input, which is input_names[0]
        name = self.model.input_names[0]
        # Finding the longest path from the start to every other layer
        self.levels = self.longest_path(name)
        singletons = []
        for l in range(len(self.levels)):
            if len(self.levels[l]) == 1:
                singletons.append(self.levels[l][0])
        return singletons

    def find_all_paths_util(self, u, d, visited, path, all_paths):
        # If the distance of the current path is greater than the longest path (the "level") to the destination node,
        # we know the destination node can't be a partition point
        if self.layer_level[u] > self.layer_level[d]:
            return False
        # Mark the current node as visited and store in path
        visited[u] = True
        path.append(u)

        # If current vertex is same as destination, then print
        # current path[] (because we've found a path from u to d)
        if u == d:
            exists = False
            # See if path already exists in list of paths
            for p in all_paths:
                if p == path:
                    exists = True
                    break

            if not exists:
                all_paths.append(path.copy())
        else:
            # If current vertex is not destination
            # Recur for all the vertices adjacent to this vertex
            for i in self.get_next(u):
                if not visited[i]:
                    ret = self.find_all_paths_util(i, d, visited, path, all_paths)
                    if not ret:
                        return False

        # Remove current vertex from path[] and mark it as unvisited
        path.pop()
        visited[u] = False
        return True

    # Finds all paths from 's' to 'd.' Returns false if a there exists a path from s that has a greater "level" than
    # d, otherwise returns true
    def find_all_paths(self, s, d) -> bool:
        # Mark all the vertices as not visited
        visited = {}
        for l in self.model.layers:
            visited[l.name] = False

        # Create an array to store paths
        path = []
        all_paths = []

        # Call the recursive helper function to find all paths
        return self.find_all_paths_util(s, d, visited, path, all_paths)

    def partitions_util(self, prev, singleton_nodes, partitions):
        # Reached the end of the model and found all the partitions
        if len(singleton_nodes) == 0:
            return partitions
        p = False
        i = -1  # So first i starts at 0
        # Starting from the previous partition point, we iterate through all the subsequent singleton nodes to find
        # the next partition point
        while not p:
            i += 1
            p = self.find_all_paths(prev, singleton_nodes[i])

        partitions.append(singleton_nodes[i])
        return self.partitions_util(singleton_nodes[i], singleton_nodes[i + 1:], partitions)

    def find_partitions(self) -> List[str]:
        inpt = self.model.input_names[0]
        return self.partitions_util(inpt, self.find_singletons(), [])

    def keras_model_memory_usage_in_bytes(self, model, batch_size: int):
        """
        Return the estimated memory usage of a given Keras model in bytes.
        This includes the model weights and layers, but excludes the dataset.

        The model shapes are multiplied by the batch size, but the weights are not.

        Args:
            model: A Keras model.
            batch_size: The batch size you intend to run the model with. If you
                have already specified the batch size in the model itself, then
                pass `1` as the argument here.
        Returns:
            An estimate of the Keras model's memory usage in bytes.

        """
        default_dtype = tf.keras.backend.floatx()
        shapes_mem_count = 0
        internal_model_mem_count = 0
        for layer in model.layers:
            if isinstance(layer, tf.keras.Model):
                internal_model_mem_count += self.keras_model_memory_usage_in_bytes(
                    layer, batch_size=batch_size
                )
            single_layer_mem = tf.as_dtype(layer.dtype or default_dtype).size
            out_shape = layer.output_shape
            if isinstance(out_shape, list):
                out_shape = out_shape[0]
            for s in out_shape:
                if s is None:
                    continue
                single_layer_mem *= s
            shapes_mem_count += single_layer_mem

        trainable_count = sum(
            [tf.keras.backend.count_params(p) for p in model.trainable_weights]
        )
        non_trainable_count = sum(
            [tf.keras.backend.count_params(p) for p in model.non_trainable_weights]
        )

        total_memory = (
                batch_size * shapes_mem_count
                + internal_model_mem_count
                + trainable_count
                + non_trainable_count
        )
        return total_memory

    def keras_layer_memory(self, layer_name, batch_size: int):
        default_dtype = tf.keras.backend.floatx()
        shapes_mem_count = 0
        internal_model_mem_count = 0

        if isinstance(layer_name, tf.keras.Model):
            internal_model_mem_count += self.keras_model_memory_usage_in_bytes(
                layer_name, batch_size=batch_size
            )
        single_layer_mem = tf.as_dtype(layer_name.dtype or default_dtype).size
        out_shape = layer_name.output_shape
        if isinstance(out_shape, list):
            out_shape = out_shape[0]
        for s in out_shape:
            if s is None:
                continue
            single_layer_mem *= s
        shapes_mem_count += single_layer_mem

        trainable_count = sum(
            [tf.keras.backend.count_params(p) for p in layer_name.trainable_weights]
        )
        non_trainable_count = sum(
            [tf.keras.backend.count_params(p) for p in layer_name.non_trainable_weights]
        )

        total_memory = (
                batch_size * shapes_mem_count
                + internal_model_mem_count
                + trainable_count
                + non_trainable_count
        )
        return total_memory

    def find_partition_memory(self, partition_points):
        part_mems = []
        # Each index represents the memory between that part pt and the next one
        for i in range(1, len(partition_points)):
            # Going backwards along layers within partition to find total memory usage
            start = self.layer_level[partition_points[i]]
            end = self.layer_level[partition_points[i - 1]]
            mem = 0
            for j in range(start, end, -1):
                for l in self.levels[j]:
                    layer_mem = self.keras_layer_memory(self.model.get_layer(l), 1)
                    mem += layer_mem
            part_mems.append(mem)
        # Nothing used after last partition pt, which is output layer
        part_mems.append(0)
        return part_mems

    # Returns transfer size of partition in Mbits
    def find_partition_transfer_size(self, partition_points) -> List[int]:
        transfer_sizes = []
        input_size = 1
        # Iterate through all elements of shape tuple except first one (which is batch size)
        for s in self.model.input.get_shape()[1:]:
            input_size *= s
        # Compression ratio is ~1.44 (according to
        # https://www.researchgate.net/publication/264417607_Fixed-Rate_Compressed_Floating-Point_Arrays)
        zfp_comp_ratio = 1.44
        # input_size gives us number of bits, need to convert to bytes
        input_size_bytes = (input_size * 8) / zfp_comp_ratio
        # Assuming all elements are floats, each float uses 8 bytes
        input_size_mbits = (input_size_bytes * 8) / (1024 ** 2)

        # Put input size as first element of transfer size array
        transfer_sizes.append(input_size_mbits)

        for i in range(len(partition_points)):
            num_outbound = len(self.model.get_layer(partition_points[i]).outbound_nodes)

            # Iterate through all elements of shape tuple except first one (which is batch size)
            output_size = 1
            for s in self.model.get_layer(partition_points[i]).get_output_at(0).get_shape()[1:]:
                output_size *= s

            # Assuming all elements are floats, each float uses 8 bytes
            output_size_bytes = (output_size * 8) / zfp_comp_ratio
            output_size_mbits = (output_size_bytes * 8) / (1024 ** 2)
            # All outputs of the layer are the same size, the total size will be (output size * num_output_nodes)
            transfer_size = num_outbound * output_size_mbits
            transfer_sizes.append(transfer_size)

        return transfer_sizes
