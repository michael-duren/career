import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def solve(text, k):
    counts = {}
    repeats = 0
    good = 0
    emit("start", right="-", window="", repeats=repeats, good=good)
    for right, letter in enumerate(text):
        counts[letter] = counts.get(letter, 0) + 1
        if counts[letter] == 2:
            repeats += 1
        if right >= k:
            old = text[right - k]
            counts[old] -= 1
            if counts[old] == 1:
                repeats -= 1
        if right >= k - 1:
            if repeats == 0:
                good += 1
            emit("window", right=right, window=text[right - k + 1:right + 1], repeats=repeats, good=good)

    emit("done", right="-", window="", repeats=repeats, good=good)
    return good


def main():
    text, k = sys.stdin.read().split()
    result(str(solve(text, int(k))))


main()
