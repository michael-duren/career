import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def digit_sum(n, depth):
    emit("call", n=n, depth=depth, result="-")
    if n < 10:
        emit("base", n=n, depth=depth, result=n)
        return n
    total = digit_sum(n // 10, depth + 1) + n % 10
    emit("return", n=n, depth=depth, result=total)
    return total


def solve(n):
    return digit_sum(n, 0)


def main():
    result(str(solve(int(sys.stdin.read()))))


main()
