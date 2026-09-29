import json
import sys

def emit(event, i, current, state, answer):
    values = [('i', i), ('current', current), ('state', state), ('answer', answer)]
    print(json.dumps({'event': event, 'variables': [{'name': name, 'value': str(value)} for name, value in values]}, separators=(',', ':')))

def result(answer):
    print(json.dumps({'result': answer}, separators=(',', ':')))

def digit_square(value):
    total = 0
    while value:
        value, digit = divmod(value, 10)
        total += digit * digit
    return total

value = int(sys.stdin.read().strip())
seen = set()
step = 0
emit("start", -1, value, 'seen=0', 'pending')
while value != 1 and value not in seen:
    seen.add(value)
    value = digit_square(value)
    emit("step", step, value, 'seen='+str(len(seen)), 'pending')
    step += 1
answer = 'true' if value == 1 else 'false'
emit("done", step, value, 'seen='+str(len(seen)), answer)
result(answer)
