import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def solve(start):
    slow = step(start)
    fast = step(step(start))
    emit("first-move", slow=slow, fast=fast)
    while slow != fast:
        slow = step(slow)
        fast = step(step(fast))
        emit("move", slow=slow, fast=fast)

    emit("meet", slow=slow, fast=fast)
    return slow


def step(x):
    return (x * x + 1) % 10


def main():
    result(str(solve(int(sys.stdin.read()))))


main()
