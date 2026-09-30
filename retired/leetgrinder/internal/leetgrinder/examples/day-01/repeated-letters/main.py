import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def solve(word):
    counts = [0] * 26
    emit("start", i="-", letter="-", count="-", repeated="")
    for i, letter in enumerate(word):
        counts[ord(letter) - ord("a")] += 1
        emit("count", i=i, letter=letter, count=counts[ord(letter) - ord("a")], repeated="")
    repeated = ""
    for k in range(26):
        if counts[k] > 1:
            repeated += chr(ord("a") + k)
            emit("repeat", i="-", letter=chr(ord("a") + k), count=counts[k], repeated=repeated)

    emit("done", i="-", letter="-", count="-", repeated=repeated)
    return repeated or "none"


def main():
    words = sys.stdin.read().split()
    result(solve(words[0] if words else ""))


main()
