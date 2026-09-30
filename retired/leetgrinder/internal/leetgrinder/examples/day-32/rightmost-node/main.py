import json
import sys
from collections import deque

def compact(value):
    return json.dumps(value, separators=(",", ":"))

def emit(event, depth, current, queue, level, result):
    values = [("depth", str(depth)), ("current", str(current)),
              ("queue", compact(list(queue))), ("level", compact(level)),
              ("result", compact(result))]
    print(compact({"event": event, "variables":
                   [{"name": name, "value": value} for name, value in values]}))

def solve(rows):
    queue = deque([0]) if rows else deque()
    depth = 0
    result = []  # MODE: level
    emit("init", depth, -1, queue, [], result)
    while queue:
        depth += 1
        count = len(queue)
        level = []
        emit("level_start", depth, -1, queue, level, result)
        for _ in range(count):
            node = queue.popleft()
            value, left, right = rows[node]
            level.append(value)
            if left != -1: queue.append(left)
            if right != -1: queue.append(right)
            emit("visit", depth, node, queue, level, result)
        result.append(level[-1])  # COMMIT
        emit("level_end", depth, -1, queue, level, result)
    emit("done", depth, -1, queue, [], result)
    print(compact({"result": compact(result)}))

def main():
    data = list(map(int, sys.stdin.read().split()))
    n = data[0]
    solve([tuple(data[1 + 3*i:4 + 3*i]) for i in range(n)])

if __name__ == "__main__":
    main()
