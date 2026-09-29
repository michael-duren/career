import json
import sys


def shown(counts):
    return "{" + ",".join(
        f"{chr(i + ord('a'))}:{count}"
        for i, count in enumerate(counts) if count != 0
    ) + "}"


def emit(event, **state):
    variables = [{"name": key, "value": str(value)} for key, value in state.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def main():
    first = sys.stdin.readline().rstrip("\n")
    second = sys.stdin.readline().rstrip("\n")
    counts = [0] * 26
    emit("start", first=first, second=second, counts=shown(counts))
    for i, letter in enumerate(first):
        counts[ord(letter) - ord("a")] += 1
        emit("add", i=i, letter=letter, counts=shown(counts))
    answer = len(first) == len(second)
    if answer:
        for i, letter in enumerate(second):
            counts[ord(letter) - ord("a")] -= 1
            emit("remove", i=i, letter=letter, counts=shown(counts))
            if counts[ord(letter) - ord("a")] < 0:
                answer = False
                break
    emit("done", counts=shown(counts), anagram=str(answer).lower())
    print(json.dumps({"result": str(answer).lower()}, separators=(",", ":")))


if __name__ == "__main__":
    main()
