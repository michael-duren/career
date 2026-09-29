import json
import sys

def emit(event, i, current, state, answer):
    values = [('i', i), ('current', current), ('state', state), ('answer', answer)]
    print(json.dumps({'event': event, 'variables': [{'name': name, 'value': str(value)} for name, value in values]}, separators=(',', ':')))

def result(answer):
    print(json.dumps({'result': answer}, separators=(',', ':')))

board = sys.stdin.read().split()
clues = []
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
        for pr, pc, previous in clues:
            if previous == digit and (pr == r or pc == c or (pr//3 == r//3 and pc//3 == c//3)):
                answer = 'false'
                break
        current = f'{digit}@{r},{c}'
        if answer == 'false':
            emit("conflict", step, current, 'checked='+str(len(clues)), answer)
            break
        clues.append((r, c, digit))
        emit("check", step, current, 'checked='+str(len(clues)), 'pending')
        step += 1
emit("done", step, '-', 'checked='+str(len(clues)), answer)
result(answer)
