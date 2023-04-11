.PHONY: all system_init node_bw partitioner deploy_pods inference_io inference_runtime dispatch_inference process_inference clean

# Change to your own container registry
REGISTRY = localhost:5000
# Change to your custom version
VERSION = latest

all: system_init node_bw partitioner deploy_pods inference_io inference_runtime dispatch_inference process_inference

system_init:
	docker build -f ./system_init_job/get_system_info_container/get_system_info.Dockerfile -t $(REGISTRY)/get_system_info:$(VERSION) .
	docker push $(REGISTRY)/get_system_info:$(VERSION)

node_bw:
	docker build -f ./system_init_job/get_node_bandwidths_container/node_iperf_connections.Dockerfile -t $(REGISTRY)/get_node_bandwidths:$(VERSION) .
	docker push $(REGISTRY)/get_node_bandwidths:$(VERSION)

partitioner:
	docker build -f ./dispatcher_pod/config_step/partitioning_container/dispatcher_partitioner.Dockerfile -t $(REGISTRY)/dispatcher_partitioner:$(VERSION) .
	docker push $(REGISTRY)/dispatcher_partitioner:$(VERSION)

deploy_pods:
	docker build -f ./dispatcher_pod/config_step/deploy_container/dispatcher_deploy.Dockerfile -t $(REGISTRY)/deploy_pods:$(VERSION) .
	docker push $(REGISTRY)/deploy_pods:$(VERSION)

inference_io:
	docker build -f ./inference_pod/io_container/pod_inference_io.Dockerfile -t $(REGISTRY)/pod_inference_io:$(VERSION) .
	docker push $(REGISTRY)/pod_inference_io:$(VERSION)

inference_runtime:
	docker build -f ./inference_pod/inference_container/inference_runtime.Dockerfile -t $(REGISTRY)/inference_runtime:$(VERSION) .
	docker push $(REGISTRY)/inference_runtime:$(VERSION)

dispatch_inference:
	docker build -f ./dispatcher_pod/inference_step/io_container/dispatcher_inference_io.Dockerfile -t $(REGISTRY)/dispatcher_inference_io:$(VERSION) .
	docker push $(REGISTRY)/dispatcher_inference_io:$(VERSION)

process_inference:
	docker build -f ./dispatcher_pod/inference_step/input_processing_container/process_input.Dockerfile -t $(REGISTRY)/process_inference_input:$(VERSION) .
	docker push $(REGISTRY)/process_inference_input:$(VERSION)

