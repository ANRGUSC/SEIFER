from io_pipes import process_input, process_output
from queue import Queue
from threading import Thread
import numpy as np
from keras.applications.resnet import preprocess_input, decode_predictions

FIFO_PATH = "/io"
FIFO_INPUT_PATH = f"{FIFO_PATH}/to_processing"
FIFO_OUTPUT_PATH = f"{FIFO_PATH}/from_processing"

input_queue = Queue(10 ** 5)
finished_queue = Queue(10 ** 5)

# Random pixels, just to get a sample output
x = np.random.random_sample((224, 224))
x = np.expand_dims(x, axis=0)
x = preprocess_input(x)


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


def print_finished_inference(q: Queue):
    print("Getting output from system")
    for i in range(100):
        print(decode_predictions(q.get(), top=1)[0])


def put_inference_input(q: Queue):
    print("Putting input into system")
    for i in range(100):
        q.put(x)


print("Running process-inference-input")
run()
