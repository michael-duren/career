import json
import sys

def emit(event, i, current, state, answer):
    values = [('i', i), ('current', current), ('state', state), ('answer', answer)]
    print(json.dumps({'event': event, 'variables': [{'name': name, 'value': str(value)} for name, value in values]}, separators=(',', ':')))

def result(answer):
    print(json.dumps({'result': answer}, separators=(',', ':')))

def show(values):
    return ','.join(map(str, values)) if values else 'empty'

tokens = list(map(int, sys.stdin.read().split()))
n = tokens[0]
first = tokens[1:1+n]
m = tokens[1+n]
second = tokens[2+n:2+n+m]
remaining = {}
for value in first:
    remaining[value] = remaining.get(value, 0) + 1
out = []
emit("start", -1, '-', 'remaining4='+str(remaining.get(4, 0)), 'empty')
for i, value in enumerate(second):
    if remaining.get(value, 0) > 0:
        remaining[value] -= 1
        out.append(value)
    emit("step", i, value, 'remaining4='+str(remaining.get(4, 0)), show(out))
answer = show(out)
emit("done", len(second), '-', 'remaining4='+str(remaining.get(4, 0)), answer)
result(answer)
