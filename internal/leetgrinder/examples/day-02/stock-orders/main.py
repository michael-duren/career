import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def solve(stock, orders):
    counts = [0] * 26
    for item in stock:
        counts[ord(item) - ord("a")] += 1
    emit("stock", i="-", item="-", left="-", counts=show_letters(counts))
    for i, item in enumerate(orders):
        counts[ord(item) - ord("a")] -= 1
        left = counts[ord(item) - ord("a")]
        emit("order", i=i, item=item, left=left, counts=show_letters(counts))
        if left < 0:
            return f"short at {i}"

    emit("done", i="-", item="-", left="-", counts=show_letters(counts))
    return "filled"


def main():
    lines = sys.stdin.read().split("\n")
    result(solve(lines[0].strip(), lines[1].strip()))


def show_letters(counts):
    return "{" + ",".join(f"{chr(ord('a') + k)}:{counts[k]}" for k in range(26) if counts[k]) + "}"


main()
