import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def show_stack(stack):
    return "[" + ",".join(stack) + "]"

def solve(tokens):
    text, redo = [], []
    emit("start", token="-", stack=show_stack(text), redo=show_stack(redo))
    for token in tokens:
        if token == "undo":
            if text:
                redo.append(text.pop())
            emit("undo", token=token, stack=show_stack(text), redo=show_stack(redo))
        elif token == "redo":
            if redo:
                text.append(redo.pop())
            emit("redo", token=token, stack=show_stack(text), redo=show_stack(redo))
        else:
            text.append(token)
            redo.clear()
            emit("type", token=token, stack=show_stack(text), redo=show_stack(redo))

    emit("done", token="-", stack=show_stack(text), redo=show_stack(redo))
    return "".join(text) or "(empty)"


def main():
    result(solve(sys.stdin.read().split()))


main()
