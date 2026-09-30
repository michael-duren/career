import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def binary(n, depth):
    emit("call", n=n, depth=depth, result="-")
    if n < 2:
        emit("base", n=n, depth=depth, result=str(n))
        return str(n)
    text = binary(n // 2, depth + 1) + str(n % 2)
    emit("return", n=n, depth=depth, result=text)
    return text


def solve(n):
    return binary(n, 0)


def main():
    result(solve(int(sys.stdin.read())))


main()
