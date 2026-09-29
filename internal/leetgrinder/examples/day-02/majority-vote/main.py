import json
import sys


def emit(event, *fields):
    variables = [{"name": fields[i], "value": str(fields[i+1])} for i in range(0, len(fields), 2)]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))

def result(value):
    print(json.dumps({"result": str(value)}, separators=(",", ":")))

def solve(values):
    candidate = 0
    balance = 0
    emit("start", "i", -1, "value", "-", "candidate", "-", "balance", 0, "seen", 0)
    for i, value in enumerate(values):
        if balance == 0:
            candidate = value
            balance = 1
        elif candidate == value:
            balance += 1
        else:
            balance -= 1
        emit("vote", "i", i, "value", value, "candidate", candidate, "balance", balance, "seen", 0)
    seen = 0
    for i, value in enumerate(values):
        if value == candidate:
            seen += 1
        emit("verify", "i", i, "value", value, "candidate", candidate, "balance", balance, "seen", seen)
    answer = str(candidate) if seen > len(values)//2 else "none"
    emit("done", "i", len(values), "value", "-", "candidate", candidate, "balance", balance, "seen", seen)
    result(answer)

if __name__ == "__main__":
    data = list(map(int, sys.stdin.read().split()))
    solve(data[1:1+data[0]])
