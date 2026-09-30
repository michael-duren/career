import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def solve(values, target):
    write = 0
    emit("start", read="-", write=write, values=show(values))
    for read in range(len(values)):
        if values[read] != target:
            values[write] = values[read]
            write += 1
            emit("keep", read=read, write=write, values=show(values))
        else:
            emit("drop", read=read, write=write, values=show(values))

    emit("done", read="-", write=write, values=show(values))
    return values[:write]


def main():
    tokens = [int(token) for token in sys.stdin.read().split()]
    n, target = tokens[0], tokens[1]
    values = tokens[2:2 + n]
    result(show(solve(values, target)))


main()
