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
slow = value
fast = value
step = 0
emit("start", -1, value, f'slow={slow};fast={fast}', 'pending')
while True:
    slow = digit_square(slow)
    fast = digit_square(digit_square(fast))
    emit("step", step, slow, f'slow={slow};fast={fast}', 'pending')
    step += 1
    if slow == 1 or fast == 1 or slow == fast:
        break
answer = 'true' if slow == 1 or fast == 1 else 'false'
emit("done", step, slow, f'slow={slow};fast={fast}', answer)
result(answer)
