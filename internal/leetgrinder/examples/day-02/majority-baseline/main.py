import json
import sys


def emit(event, *fields):
    variables = [{"name": fields[i], "value": str(fields[i+1])} for i in range(0, len(fields), 2)]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))

def result(value):
    print(json.dumps({"result": str(value)}, separators=(",", ":")))

def solve(values):
    emit("start", "i", -1, "candidate", "-", "tally", 0, "answer", "none")
    answer = "none"
    last_seen = 0
    for i, candidate in enumerate(values):
        seen = 0
        for value in values:
            if value == candidate:
                seen += 1
        last_seen = seen
        emit("test", "i", i, "candidate", candidate, "tally", seen, "answer", "none")
        if seen > len(values)//2:
            answer = str(candidate)
            break
    emit("done", "i", len(values), "candidate", str(candidate), "tally", last_seen, "answer", answer)
    result(answer)

if __name__ == "__main__":
    data = list(map(int, sys.stdin.read().split()))
    solve(data[1:1+data[0]])
