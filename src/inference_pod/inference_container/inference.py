import os
import queue
from threading import Thread
import tensorflow as tf
from queue import Queue
from io_pipes import process_input, process_output

NODE = os.getenv("NODE")
#CONFIG_DIRECTORY = f"/nfs/model_config/partitions/{NODE}"
FIFO_PATH = "/io"
FIFO_INPUT_PATH = f"{FIFO_PATH}/to_inference"
FIFO_OUTPUT_PATH = f"{FIFO_PATH}/from_inference"

# interpreter = tf.lite.Interpreter(model_path=f"{CONFIG_DIRECTORY}/model.tflite")
# interpreter.allocate_tensors()
# input_index = interpreter.get_input_details()[0]["index"]
# output_index = interpreter.get_output_details()[0]["index"]
# print("Inference runtime set up")

input_q = queue.Queue(10 ** 5)
output_q = queue.Queue(10 ** 5)

def run():
    print("Started running inference runtime")

    inf = Thread(target=inference, args=(input_q, output_q))
    inpt = Thread(target=process_input, args=(FIFO_INPUT_PATH, input_q))
    outpt = Thread(target=process_output, args=(FIFO_OUTPUT_PATH, output_q))
    inf.start()
    inpt.start()
    outpt.start()
    inf.join()
    inpt.join()
    outpt.join()


def inference(input_q: Queue, output_q: Queue):
    while True:
        inpt = input_q.get()

        # interpreter.set_tensor(input_index, inpt)
        # interpreter.invoke()
        # prediction = interpreter.get_tensor(output_index)

        #print("Got prediction", list(prediction))
        #output_q.put(prediction)
        print("Got from input queue: ", str(inpt))
        output_q.put(inpt)


run()
