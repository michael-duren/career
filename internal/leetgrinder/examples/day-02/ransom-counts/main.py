import json
import sys


def emit(event, *fields):
    variables = [{"name": fields[i], "value": str(fields[i+1])} for i in range(0, len(fields), 2)]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))

def result(value):
    print(json.dumps({"result": str(value)}, separators=(",", ":")))

def state_counts(counts):
    return "{" + ",".join(f"{chr(i+97)}:{n}" for i,n in enumerate(counts) if n) + "}"

def solve(note, magazine):
    counts = [0] * 26
    emit("start", "i", -1, "letter", "-", "position", -1, "state", "{}", "ok", "true")
    for j, letter in enumerate(magazine):
        counts[ord(letter)-97] += 1
        emit("supply", "i", j, "letter", letter, "position", j, "state", state_counts(counts), "ok", "true")
    ok = True
    for i, letter in enumerate(note):
        counts[ord(letter)-97] -= 1
        ok = counts[ord(letter)-97] >= 0
        emit("request", "i", i, "letter", letter, "position", -1, "state", state_counts(counts), "ok", str(ok).lower())
        if not ok:
            break
    emit("done", "i", len(note), "letter", "-", "position", -1, "state", state_counts(counts), "ok", str(ok).lower())
    result(str(ok).lower())

if __name__ == "__main__":
    note, magazine = sys.stdin.read().split()
    solve(note, magazine)
