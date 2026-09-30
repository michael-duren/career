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
    head = build(values)
    lead = trail = head
    for step in range(target):
        lead = lead.next
    emit("gap", lead="-" if lead is None else lead.value, trail=trail.value)
    while lead:
        lead = lead.next
        trail = trail.next
        emit("move", lead="-" if lead is None else lead.value, trail=trail.value)

    emit("done", lead="-", trail=trail.value)
    return str(trail.value)


def main():
    tokens = [int(token) for token in sys.stdin.read().split()]
    n, target = tokens[0], tokens[1]
    values = tokens[2:2 + n]
    result(solve(values, target))


main()
