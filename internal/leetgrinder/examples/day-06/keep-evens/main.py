import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def solve(values):
    write = 0
    emit("start", read="-", write=write, values=show(values))
    for read in range(len(values)):
        if values[read] % 2 == 0:
            values[write] = values[read]
            write += 1
            emit("keep", read=read, write=write, values=show(values))
        else:
            emit("skip", read=read, write=write, values=show(values))

    emit("done", read="-", write=write, values=show(values))
    return values[:write]


def main():
    tokens = [int(token) for token in sys.stdin.read().split()]
    n = tokens[0]
    result(show(solve(tokens[1:1 + n])))


main()
