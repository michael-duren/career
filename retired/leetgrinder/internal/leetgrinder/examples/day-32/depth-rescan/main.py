import json
import sys

def compact(value):
    return json.dumps(value, separators=(",", ":"))

def emit(event, depth, visits, level, result):
    values = [("depth", str(depth)), ("visits", str(visits)),
              ("level", compact(level)), ("result", compact(result))]
    print(compact({"event": event, "variables":
                   [{"name": name, "value": value} for name, value in values]}))

def solve(rows):
    def height(node):
        if node == -1: return 0
        _, left, right = rows[node]
        return 1 + max(height(left), height(right))

    def collect(node, remaining, level):
        if node == -1: return 0
        value, left, right = rows[node]
        if remaining == 1:
            level.append(value)
            return 1
        return 1 + collect(left, remaining - 1, level) + collect(right, remaining - 1, level)

    result = []  # MODE: rescan
    visits = 0
    emit("init", 0, visits, [], result)
    maximum = height(0) if rows else 0
    emit("height", maximum, visits, [], result)
    for depth in range(1, maximum + 1):
        level = []
        emit("pass_start", depth, visits, level, result)
        visits += collect(0, depth, level)
        result.append(level)
        emit("pass_end", depth, visits, level, result)
    emit("done", maximum, visits, [], result)
    print(compact({"result": compact(result)}))

def main():
    data = list(map(int, sys.stdin.read().split()))
    n = data[0]
    solve([tuple(data[1 + 3*i:4 + 3*i]) for i in range(n)])

if __name__ == "__main__":
    main()
