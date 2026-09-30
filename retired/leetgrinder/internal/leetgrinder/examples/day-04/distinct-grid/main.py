import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def solve(grid):
    n = len(grid)
    rows = [set() for _ in range(n)]
    cols = [set() for _ in range(n)]
    emit("start", r="-", c="-", letter="-", row="-", col="-")
    for r in range(n):
        for c in range(n):
            letter = grid[r][c]
            if letter == ".":
                continue
            if letter in rows[r] or letter in cols[c]:
                emit("clash", r=r, c=c, letter=letter, row=show_set(rows[r]), col=show_set(cols[c]))
                return "false"
            rows[r].add(letter)
            cols[c].add(letter)
            emit("add", r=r, c=c, letter=letter, row=show_set(rows[r]), col=show_set(cols[c]))

    emit("done", r="-", c="-", letter="-", row="-", col="-")
    return "true"


def main():
    tokens = sys.stdin.read().split()
    result(solve(tokens[1:1 + int(tokens[0])]))


def show_set(letters):
    return "{" + ",".join(sorted(letters)) + "}"


main()
