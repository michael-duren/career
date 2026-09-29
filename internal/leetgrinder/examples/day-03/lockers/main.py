import json
import sys


def emit(event, **values):
    variables = [{"name": name, "value": str(value)} for name, value in values.items()]
    print(json.dumps({"event": event, "variables": variables}, separators=(",", ":")))


def result(answer):
    print(json.dumps({"result": answer}, separators=(",", ":")))


def show(values):
    return "[" + ",".join(str(value) for value in values) + "]"


def solve(pairs):
    locker_of = {}
    student_of = {}
    emit("start", student="-", locker="-", lockerOf=show_map(locker_of), studentOf=show_map(student_of))
    for student, locker in pairs:
        if student in locker_of and locker_of[student] != locker:
            emit("student-conflict", student=student, locker=locker, lockerOf=show_map(locker_of), studentOf=show_map(student_of))
            return "false"
        if locker in student_of and student_of[locker] != student:
            emit("locker-conflict", student=student, locker=locker, lockerOf=show_map(locker_of), studentOf=show_map(student_of))
            return "false"
        locker_of[student] = locker
        student_of[locker] = student
        emit("assign", student=student, locker=locker, lockerOf=show_map(locker_of), studentOf=show_map(student_of))

    emit("done", student="-", locker="-", lockerOf=show_map(locker_of), studentOf=show_map(student_of))
    return "true"


def main():
    tokens = sys.stdin.read().split()
    n = int(tokens[0])
    result(solve([(tokens[1 + 2 * k], tokens[2 + 2 * k]) for k in range(n)]))


def show_map(mapping):
    return "{" + ",".join(f"{key}:{mapping[key]}" for key in sorted(mapping)) + "}"


main()
