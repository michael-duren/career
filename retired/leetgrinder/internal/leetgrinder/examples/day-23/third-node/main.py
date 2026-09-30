import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


class Node:
    def __init__(self, value, next=None):
        self.value = value
        self.next = next


def build(values):
    head = None
    for value in reversed(values):
        head = Node(value, head)
    return head


def render(node):
    parts = []
    while node:
        parts.append(str(node.value))
        node = node.next
    return "->".join(parts) or "empty"

def solve(values, target):
    slow = fast = build(values)
    emit("start", slow=slow.value, fast=fast.value)
    while fast.next and fast.next.next and fast.next.next.next:
        slow = slow.next
        fast = fast.next.next.next
        emit("move", slow=slow.value, fast=fast.value)

    emit("done", slow=slow.value, fast=fast.value)
    return str(slow.value)


def main():
    tokens = [int(token) for token in sys.stdin.read().split()]
    n, target = tokens[0], tokens[1]
    values = tokens[2:2 + n]
    result(solve(values, target))


main()
