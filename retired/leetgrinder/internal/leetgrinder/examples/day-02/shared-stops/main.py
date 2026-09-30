import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def solve(first, second):
    stops = set()
    position = 0
    for length in first:
        position += length
        stops.add(position)
        emit("stop", route="A", length=length, position=position, shared="[]")
    shared = []
    position = 0
    for length in second:
        position += length
        if position in stops:
            shared.append(position)
        emit("check", route="B", length=length, position=position, shared=show(shared))

    emit("done", route="-", length="-", position="-", shared=show(shared))
    return shared


def main():
    lines = sys.stdin.read().strip().split("\n")
    first = [int(x) for x in lines[0].split()]
    second = [int(x) for x in lines[1].split()]
    result(show(solve(first, second)))


main()
