import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def solve(a, b):
    i, j, carry = len(a) - 1, len(b) - 1, 0
    digits = []
    emit("start", i=i, j=j, carry=carry, digits=show(digits))
    while i >= 0 or j >= 0 or carry:
        total = carry
        if i >= 0:
            total += a[i]
        if j >= 0:
            total += b[j]
        digits.append(total % 10)
        carry = total // 10
        emit("column", i=i, j=j, carry=carry, digits=show(digits))
        i -= 1
        j -= 1

    digits.reverse()
    emit("done", i=i, j=j, carry=carry, digits=show(digits))
    return digits


def main():
    lines = sys.stdin.read().strip().split("\n")
    a = [int(x) for x in lines[0].split()]
    b = [int(x) for x in lines[1].split()]
    result(show(solve(a, b)))


main()
