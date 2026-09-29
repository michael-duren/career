import json
import sys

def emit(event, i, current, state, answer):
    values = [('i', i), ('current', current), ('state', state), ('answer', answer)]
    print(json.dumps({'event': event, 'variables': [{'name': name, 'value': str(value)} for name, value in values]}, separators=(',', ':')))

def result(answer):
    print(json.dumps({'result': answer}, separators=(',', ':')))

board = sys.stdin.read().split()
rows = [set() for _ in range(9)]
cols = [set() for _ in range(9)]
boxes = [set() for _ in range(9)]
step = 0
answer = 'true'
emit("start", -1, '-', 'checked=0', 'pending')
for r in range(9):
    if answer == 'false':
        break
    for c in range(9):
        digit = board[r][c]
        if digit == '.':
            continue
        box = (r//3)*3 + c//3
        current = f'{digit}@{r},{c}'
        if digit in rows[r] or digit in cols[c] or digit in boxes[box]:
            answer = 'false'
            emit("conflict", step, current, 'checked='+str(step), answer)
            break
        rows[r].add(digit)
        cols[c].add(digit)
        boxes[box].add(digit)
        step += 1
        emit("check", step-1, current, 'checked='+str(step), 'pending')
emit("done", step, '-', 'checked='+str(step), answer)
result(answer)
