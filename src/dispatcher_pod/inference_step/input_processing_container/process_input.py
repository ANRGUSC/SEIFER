import json
import os.path
import queue
import time

from io_pipes import process_input, process_output
from queue import Queue
from threading import Thread
import numpy as np
from keras.utils import load_img, img_to_array
from keras.applications.resnet import preprocess_input

IO_DIR = "/io"
FIFO_INPUT_PATH = f"{IO_DIR}/to_processing"
FIFO_OUTPUT_PATH = f"{IO_DIR}/from_processing"
READINESS_ENDPOINT = f"{IO_DIR}/readiness_check/ready.txt"

DATA_NUM = 100

# Limit queue size to prevent memory blowup
input_queue = Queue(10)
finished_queue = Queue(10)

# Test image
img = load_img("elephant.jpg", target_size=(224, 224))
x = img_to_array(img)
dims_exp = np.expand_dims(x, axis=0)
arr = preprocess_input(dims_exp)

# Queue to notify print_finished_inference() when to start
start_q = queue.Queue(1)


def run():
    print("Started run()")
    put_inference_inpt = Thread(target=put_inference_input, args=(input_queue, start_q))
    process_outpt = Thread(target=process_output, args=(FIFO_OUTPUT_PATH, input_queue))

    process_inpt = Thread(target=process_input, args=(FIFO_INPUT_PATH, finished_queue))
    print_inference_output = Thread(target=print_finished_inference, args=(finished_queue, start_q))

    process_inpt.start()
    process_outpt.start()
    put_inference_inpt.start()
    print_inference_output.start()

    process_inpt.join()
    process_outpt.join()
    put_inference_inpt.join()
    print_inference_output.join()


time_min = 10
time_sec = time_min * 60


def print_finished_inference(finished_q: Queue, start_time_q: Queue):
    print("Getting output from system")
    # for i in range(DATA_NUM):
    #     # print(f"Output #{i+1}: {decode_predictions(q.get(block=True), top=1)[0]}")
    #     q.get(block=True)
    #     print(f"Output #{i + 1}")
    res_count = 0
    end_to_end = 0
    # Wait till the put_inference_input() thread has started and get the start time from there
    start = start_time_q.get(block=True)
    while (time.time() - start) < time_sec:
        res = finished_q.get(block=True)
        res_count += 1
        print(f"Got result #{res_count}")
        if res_count == 1:
            end_to_end = time.time() - start
    print(f"{res_count} results in {time_min} min")
    print(f"Throughput (inf/sec): {res_count / time_sec}")
    print(f"End-to-End Latency (sec): {end_to_end}")
    exit(0)


def put_inference_input(inference_q: Queue, start_time_q: Queue):
    print("Putting input into system")
    # Wait for dispatch-inference-data runtime to be set up
    while not (os.path.exists(READINESS_ENDPOINT)):
        print("Waiting for readiness endpoint to exist")
        time.sleep(0.5)
    # Make sure the file actually has the right ready status
    with open(READINESS_ENDPOINT, 'r') as f:
        data = json.load(f)
        while not data['ready']:
            print("Waiting for readiness check to succeed")
            time.sleep(0.5)
    print("System ready, sending inference data")
    # Put inference data into system
    start = time.time()
    start_time_q.put(start, block=True)
    for i in range(DATA_NUM):
        inference_q.put(arr, block=True)


print("Running process-inference-input")
run()
