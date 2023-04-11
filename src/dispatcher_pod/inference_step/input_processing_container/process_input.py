import json
import os.path
import time

from io_pipes import process_input, process_output
from queue import Queue
from threading import Thread
import numpy as np
from keras.utils import load_img, img_to_array
from keras.applications.resnet import preprocess_input, decode_predictions

FIFO_PATH = "/io"
FIFO_INPUT_PATH = f"{FIFO_PATH}/to_processing"
FIFO_OUTPUT_PATH = f"{FIFO_PATH}/from_processing"
READINESS_ENDPOINT = "/readiness_check/ready.txt"

input_queue = Queue(10 ** 5)
finished_queue = Queue(10 ** 5)

# Test image
img = load_img("elephant.jpg", target_size=(224, 224))
x = img_to_array(img)
dims_exp = np.expand_dims(x, axis=0)
arr = preprocess_input(dims_exp)


def run():
    print("Started run()")
    put_inference_inpt = Thread(target=put_inference_input, args=(input_queue,))
    process_outpt = Thread(target=process_output, args=(FIFO_OUTPUT_PATH, input_queue))

    process_inpt = Thread(target=process_input, args=(FIFO_INPUT_PATH, finished_queue))
    print_outpt = Thread(target=print_finished_inference, args=(finished_queue,))

    process_inpt.start()
    process_outpt.start()
    put_inference_inpt.start()
    print_outpt.start()

    process_inpt.join()
    process_outpt.join()
    put_inference_inpt.join()
    print_outpt.join()


# Create readiness probes for inference pods, saying that they're ready only when they've connected to their next and previous nodes
# Then from the dispatch-inference-data runtime, use client-go to get all the inference pod status, and wait for them to all be ready
# TODO figure out how to make python runtime know when dispatch-inference-data is ready to accept inference input

def print_finished_inference(q: Queue):
    print("Getting output from system")
    for i in range(100):
        # print(f"Output #{i+1}: {decode_predictions(q.get(block=True), top=1)[0]}")
        q.get(block=True)
        print(f"Output #{i + 1}")


def put_inference_input(q: Queue):
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
    # Put inference data into system
    for i in range(100):
        q.put(arr)


print("Running process-inference-input")
run()
