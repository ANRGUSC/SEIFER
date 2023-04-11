import os
import time


def create_pipe(path: str, mode: str) -> int:
    options = 0
    # Open pipes as blocking on the python side
    if mode == "send":
        # The writer should be in blocking mode so that it waits for the reader to open the pipe
        options = os.O_WRONLY
    elif mode == "recv":
        # The reader needs to open the pipe and be in non-blocking mode
        try:
            os.mkfifo(path, 0o666)
        except FileExistsError:
            pass
        options = os.O_NONBLOCK | os.O_RDONLY

    print("os.open() pipe")
    pipe = 0
    err = FileNotFoundError()
    while type(err) is FileNotFoundError:
        try:
            pipe = os.open(path, options)
            # break out of error loop
            err = 0
        except FileNotFoundError as e:
            print("Pipe wasn't created for", mode)
            err = e
            time.sleep(0.5)
    print("Pipe created")
    return pipe
