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
used = [False] * n
out = []
emit("start", -1, '-', 'used=0', 'empty')
for i, value in enumerate(second):
    for j, candidate in enumerate(first):
        if not used[j] and candidate == value:
            used[j] = True
            out.append(value)
            break
    emit("step", i, value, 'used='+str(sum(used)), show(out))
answer = show(out)
emit("done", len(second), '-', 'used='+str(sum(used)), answer)
result(answer)
