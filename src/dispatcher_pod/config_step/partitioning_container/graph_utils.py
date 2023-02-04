import math

import networkx as nx
import numpy as np
from itertools import combinations
from typing import Tuple, List, Dict


class GraphUtils:
    def __init__(self):
        self.path_from = {}

    def k_path_color_coding(self, graph: nx.Graph, k: int):
        # Creates speedup of algorithm
        a = 1.3
        for i in range(int(10 * (math.e ** k))):
            rng = np.random.default_rng()
            coloring = rng.integers(1, (a * k) + 1, len(graph.nodes()))
            j = 0
            for v in graph.nodes():
                graph.nodes()[v]["color"] = coloring[j]
                j += 1
            g = {}
            for v in graph.nodes():
                g[v] = {}
                g[v][frozenset([graph.nodes()[v]['color']])] = {}
                g[v][frozenset([graph.nodes()[v]['color']])]['hasPath'] = True
                g[v][frozenset([graph.nodes()[v]['color']])]['path'] = [v]
                for c in range(1, k + 1):
                    if c != graph.nodes()[v]['color']:
                        g[v][frozenset([c])] = {}
                        g[v][frozenset([c])]['hasPath'] = False
            K = range(1, k + 1)
            for s in range(1, k):
                possible_S = list(combinations(K, s))
                for u in graph.nodes():
                    for v in nx.neighbors(graph, u):
                        for S in possible_S:
                            Sset = frozenset(S)
                            if Sset in g[u] and g[u][Sset]['hasPath'] == True:
                                if graph.nodes()[v]['color'] not in Sset:
                                    newSet = list(S).copy()
                                    newSet.append(graph.nodes()[v]['color'])
                                    g[v][frozenset(newSet)] = {}
                                    g[v][frozenset(newSet)]['hasPath'] = True
                                    newPath = g[u][frozenset(S)]['path'].copy()
                                    newPath.append(v)
                                    g[v][frozenset(newSet)]['path'] = newPath

            for u in graph.nodes():
                if frozenset(K) in g[u]:
                    if g[u][frozenset(K)]['hasPath']:
                        return g[u][frozenset(K)]['path']

        return False

    def modified_k_path_color_coding(self, graph: nx.Graph, k: int, start, end):
        # Creates speedup of algorithm
        a = 1.3
        for i in range(int(10 * (math.e ** k))):
            rng = np.random.default_rng()
            coloring = rng.integers(1, (a * k) + 1, len(graph.nodes()))
            j = 0
            for v in graph.nodes():
                graph.nodes()[v]["color"] = coloring[j]
                j += 1
            g = {}
            for v in graph.nodes():
                g[v] = {}
                g[v][frozenset([graph.nodes()[v]['color']])] = {}
                g[v][frozenset([graph.nodes()[v]['color']])]['hasPath'] = True
                g[v][frozenset([graph.nodes()[v]['color']])]['path'] = [v]
                for c in range(1, k + 1):
                    if c != graph.nodes()[v]['color']:
                        g[v][frozenset([c])] = {}
                        g[v][frozenset([c])]['hasPath'] = False
            K = range(1, k + 1)
            for s in range(1, k):
                possible_S = list(combinations(K, s))
                for u in graph.nodes():
                    if start is not None and s == 1 and u != start:
                        continue
                    for v in nx.neighbors(graph, u):
                        if v is not None and v == end and s != k - 1:
                            continue
                        for S in possible_S:
                            Sset = frozenset(S)
                            if Sset in g[u] and g[u][Sset]['hasPath'] == True:
                                if graph.nodes[v]['color'] not in Sset:
                                    newSet = list(S).copy()
                                    newSet.append(graph.nodes()[v]['color'])
                                    g[v][frozenset(newSet)] = {}
                                    g[v][frozenset(newSet)]['hasPath'] = True
                                    newPath = g[u][frozenset(S)]['path'].copy()
                                    newPath.append(v)
                                    g[v][frozenset(newSet)]['path'] = newPath

            if end is not None:
                if frozenset(K) in g[end]:
                    if g[end][frozenset(K)]['hasPath']:
                        return g[end][frozenset(K)]['path']

            else:
                for u in graph.nodes():
                    if frozenset(K) in g[u]:
                        if g[u][frozenset(K)]['hasPath']:
                            return g[u][frozenset(K)]['path']

        return False

    def threshold(self, X: int, edges, classes: Dict[Tuple, int], t: int):
        for e in edges:
            name = e[2]['name']
            if e[2]['weight'] < t:
                classes[name] = X - 1
            else:
                classes[name] = X

    def subgraph_k_path(self, G: nx.Graph, X: int, k: int):
        # For the binary search we want the edge in reverse order
        edge_list = sorted(G.edges(data=True), key=lambda x: x[2]['weight'], reverse=True)

        low = 0
        high = len(edge_list)
        classes = {}
        best_path = []
        while low < high:
            median = (low + high) // 2
            med_weight = edge_list[median][2]['weight']
            self.threshold(X, edge_list, classes, med_weight)
            x_edges = [(e[0], e[1]) for e in edge_list if classes[e[2]['name']] == X]
            G_x = G.edge_subgraph(x_edges).copy()
            result = self.k_path_color_coding(G_x, k)
            if not result:
                low = median + 1
            else:
                high = median
                best_path = result

        G.remove_nodes_from(best_path)
        return best_path

    def modified_subgraph_k_path(self, G: nx.Graph, X: int, k: int, s, u):
        # For the binary search we want the edge in reverse order
        edge_list = sorted(G.edges(data=True), key=lambda x: x[2]['weight'], reverse=True)

        low = 0
        high = len(edge_list)
        classes = {}
        best_path = []
        while low < high:
            median = (low + high) // 2
            med_weight = edge_list[median][2]['weight']
            self.threshold(X, edge_list, classes, med_weight)
            x_edges = [(e[0], e[1]) for e in edge_list if classes[e[2]['name']] == X]
            G_x = G.edge_subgraph(x_edges).copy()
            if s is not None and s not in G_x:
                low = median + 1
                continue
            if u is not None and u not in G_x:
                low = median + 1
                continue
            result = self.modified_k_path_color_coding(G_x, k, s, u)
            if result == False or len(result) == 0:
                low = median + 1
            else:
                high = median
                best_path = result

        G.remove_nodes_from(best_path)
        return best_path

    def k_path_matching(self, g: nx.Graph, Q: List[str], S: List[int], C: int):
        original_graph = g.copy()
        N = [None] * len(Q)
        for X in range(C, 0, -1):
            x_paths = self.find_subarrays(S, X)
            x_paths = sorted(x_paths, key=lambda x: len(x), reverse=True)
            for j in range(len(x_paths)):
                start_idx = x_paths[j][0]
                end_idx = start_idx + len(x_paths[j])
                start_v = N[start_idx]
                end_v = N[end_idx]
                if start_v is not None and start_v not in g:
                    nodes_to_add = list(g.nodes())
                    nodes_to_add.append(start_v)
                    g = original_graph.subgraph(nodes_to_add).copy()
                if end_v is not None and end_v not in g:
                    nodes_to_add = list(g.nodes())
                    nodes_to_add.append(end_v)
                    g = original_graph.subgraph(nodes_to_add).copy()

                path = self.modified_subgraph_k_path(g, X, len(x_paths[j]) + 1, start_v, end_v)
                N[start_idx:start_idx + len(path)] = path

        return N

    def find_subarrays(self, S, X):
        x = np.array(S)
        a = x == X
        inds = [i for i in range(len(S))]
        splits = np.split(inds, np.where(np.diff(a) != 0)[0] + 1)
        subs = [s for s in splits if S[s[0]] == X]
        return subs

    def classify(self, transfer_sizes: List[int], chosen_sizes: List[int], num_bins):
        bins = np.histogram_bin_edges(transfer_sizes, bins=num_bins)
        # Returns the class that each transfer size belongs to
        classes = np.digitize(chosen_sizes, bins)
        return classes

    def create_partition_graph(self, node_capacity: int, partitions: List[str], transfer_sizes, partition_mems):
        partitions_dag = nx.DiGraph()
        for i in range(len(partitions)):
            for j in range(i + 1, len(partitions) + 1):
                mem = sum(partition_mems[i:j - 1])
                # Partition has to fit into node
                if mem < node_capacity:
                    node_name = f"{i}-{j}"
                    # End layer of partition is exclusive
                    partitions_dag.add_node(node_name, partition=(i, j))

        for n1 in partitions_dag.nodes(data=True):
            for n2 in partitions_dag.nodes(data=True):
                n1_name = n1[0]
                n2_name = n2[0]
                uEnd = n1[1]['partition'][1]
                vStart = n2[1]['partition'][0]
                if uEnd == vStart:
                    w = transfer_sizes[uEnd - 1]
                    partitions_dag.add_edge(n1_name, n2_name, weight=w)
        return partitions_dag, transfer_sizes

    def min_cost_path(self, G: nx.Graph, v):
        # Node is leaf node
        if len(G[v]) == 0:
            return [v], 0

        # Not actually the last layer, its the layer after the last
        partition_last_layer = G.nodes()[v]['partition'][1]
        if partition_last_layer not in self.path_from:
            min_path = []
            min_cost = math.inf
            for c in G[v]:
                path, cost = self.min_cost_path(G, c)
                if cost < min_cost:
                    min_cost = cost
                    min_path = path

            self.path_from[partition_last_layer] = (min_path, min_cost)

        min_path, min_cost = self.path_from[partition_last_layer]

        # The child that resulted in the min cost path
        chosen_node = min_path[0]
        # Path starting at v and going to a leaf
        new_path = [v]
        new_path.extend(min_path)
        new_cost = G[v][chosen_node]['weight'] + min_cost
        return new_path, new_cost

    def partition(self, G: nx.Graph, transfer_sizes: List, num_nodes: int, num_bins: int):
        roots = []
        for n in G.nodes():
            if G.in_degree(n) == 0:
                roots.append(n)

        min_path = []
        min_cost = math.inf
        for r in roots:
            path, cost = self.min_cost_path(G, r)
            if len(path) > num_nodes:
                continue
            if cost < min_cost:
                min_cost = cost
                min_path = path

        chosen_transfer_sizes = []
        for p in range(len(min_path) - 1):
            ts = G[min_path[p]][min_path[p + 1]]['weight']
            chosen_transfer_sizes.append(ts)

        chosen_partitions = []
        for m in min_path:
            chosen_partitions.append(G.nodes()[m]['partition'])

        transfer_size_classes = self.classify(transfer_sizes, chosen_transfer_sizes, num_bins)
        return chosen_partitions, transfer_size_classes, chosen_transfer_sizes

    def partition_and_place(self, num_nodes: int, node_capacity: int, comm_graph: nx.Graph, num_classes, partitions,
                            transfers, partition_mems):

        G_p, transfer_sizes = self.create_partition_graph(node_capacity, partitions, transfers, partition_mems)
        Q, S, transfer_size_weights = self.partition(G_p, transfer_sizes, num_nodes, num_classes)
        if len(Q) == 0:
            raise MemoryError("Can't partition with specified number of nodes and capacity")
        if len(Q) == 1:
            raise NotImplementedError("Only one partition necessary")

        G_c = comm_graph.copy()
        N = self.k_path_matching(G_c, Q, S, num_classes)
        # Rare case, usually if there's too many bandwidth classes
        if None in N:
            raise NotImplementedError("Couldn't find matching")
        return Q, N
