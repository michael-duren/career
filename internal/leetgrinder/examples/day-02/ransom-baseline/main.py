import json
import sys


def emit(event, *fields):
    variables = [{"name": fields[i], "value": str(fields[i+1])} for i in range(0, len(fields), 2)]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))

def result(value):
    print(json.dumps({"result": str(value)}, separators=(",", ":")))

def solve(note, magazine):
    used = [False] * len(magazine)
    emit("start", "i", -1, "letter", "-", "position", -1, "state", "[]", "ok", "true")
    ok = True
    for i, letter in enumerate(note):
        found = False
        for j, supply in enumerate(magazine):
            emit("try", "i", i, "letter", letter, "position", j, "state", str([k for k,v in enumerate(used) if v]), "ok", "true")
            if supply == letter and not used[j]:
                used[j] = True
                found = True
                emit("take", "i", i, "letter", letter, "position", j, "state", str([k for k,v in enumerate(used) if v]), "ok", "true")
                break
        if not found:
            ok = False
            emit("missing", "i", i, "letter", letter, "position", -1, "state", str([k for k,v in enumerate(used) if v]), "ok", "false")
            break
    emit("done", "i", len(note), "letter", "-", "position", -1, "state", str([k for k,v in enumerate(used) if v]), "ok", str(ok).lower())
    result(str(ok).lower())

if __name__ == "__main__":
    note, magazine = sys.stdin.read().split()
    solve(note, magazine)
