import errno
import os
import select
from queue import Queue
import lz4.frame
import zfpy
from create_pipes import create_pipe


def pipe_recv(pipe: int):
    size_left = 4
    byts = bytearray()
    while size_left > 0:
        try:
            recv = os.read(pipe, size_left)
        except ConnectionResetError:
            raise IOError("Connection broken")
        except BlockingIOError:
            select.select([pipe], [], [])
            continue
        size_left -= len(recv)
        byts.extend(recv)
    data_size = int.from_bytes(byts, 'big')

    data = bytearray(data_size)
    data_counter = 0
    while data_counter < data_size:
        try:
            recv = os.read(pipe, data_size - data_counter)
        except ConnectionResetError:
            raise IOError("Connection broken")
        except BlockingIOError:
            select.select([pipe], [], [])
            continue
        data[data_counter:data_counter + len(recv)] = recv
        data_counter += len(recv)
    return data


def pipe_send(pipe: int, data: bytes):
    data_size = len(data)
    size_bytes = data_size.to_bytes(4, 'big')
    while len(size_bytes) > 0:
        try:
            sent = os.write(pipe, size_bytes)
        except BrokenPipeError:
            raise IOError("Connection broken")
        except BlockingIOError:
            select.select([], [pipe], [])
            continue
        size_bytes = size_bytes[sent:]

    data_counter = 0
    while data_counter < data_size:
        try:
            sent = os.write(pipe, data[data_counter:])
        except BrokenPipeError:
            raise IOError("Connection broken")
        except BlockingIOError:
            select.select([], [pipe], [])
            continue
        data_counter += sent


# Reads data from FIFO and decompresses it
def process_input(input_path: str, q: Queue):
    print("Creating recv pipe")
    recv_pipe = create_pipe(input_path, "recv")
    while True:
        binary_data = bytearray()
        try:
            print("Waiting for data on pipe")
            binary_data = pipe_recv(recv_pipe)
            print("Received data")
        except IOError:
            print("Connection closed, recreating recv pipe")
            recv_pipe = create_pipe(input_path, "recv")
        finally:
            #arr_input = zfpy.decompress_numpy(lz4.frame.decompress(binary_data))
            print("Decompressed input")
            q.put(binary_data)
            print("Sent input through queue")


# Compresses data and sends it through FIFO
def process_output(output_path: str, q: Queue):
    print("Creating send pipe")
    send_pipe = create_pipe(output_path, "send")
    while True:
        output_arr = q.get()
        print("Got data from queue")
        #binary_output = lz4.frame.compress(zfpy.compress_numpy(output_arr))
        print("Compressed data")
        try:
            pipe_send(send_pipe, output_arr)
            print("Sent data thru pipe")
        except IOError:
            print("Connection closed, recreating send pipe")
            send_pipe = create_pipe(output_path, "send")
