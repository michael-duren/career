#!/usr/bin/env python3
"""Compile and verify authored Leetgrinder examples against their traces."""

import argparse
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path

LANGUAGES = {"cpp", "python", "java", "go"}
DEFAULT_FILES = {"cpp": ("main.cpp",), "python": ("main.py",), "java": ("Main.java",),
                 "go": ("main.go", "main.go.txt")}
ROLES = {"neutral", "active", "visited", "discarded", "result"}
VERSION_COMMANDS = {"cpp": ["g++", "--version"], "python": ["python3", "--version"],
                    "java": ["javac", "-version"], "go": ["go", "version"]}


class InvalidExample(ValueError):
    pass


def require(condition, context, message):
    if not condition:
        raise InvalidExample(f"{context}: {message}")


def nonempty(value):
    return isinstance(value, str) and bool(value.strip())


def unique(items, context):
    require(len(items) == len(set(items)), context, "duplicate IDs")


def validate_scene(scene, context):
    require(isinstance(scene, dict), context, "scene must be an object")
    width, height = scene.get("width"), scene.get("height")
    require(type(width) is int and width > 0 and type(height) is int and height > 0,
            context, "scene width/height must be positive integers")
    require(nonempty(scene.get("description")), context, "scene description is required")
    ids = []
    node_ids = set()
    for group in ("nodes", "edges", "labels"):
        entries = scene.get(group)
        require(isinstance(entries, list), context, f"scene {group} must be a list")
        for entry in entries:
            require(isinstance(entry, dict) and nonempty(entry.get("id")), context, f"scene {group} ID is required")
            ids.append(entry["id"])
            if group == "nodes":
                node_ids.add(entry["id"])
                require(entry.get("shape") in {"rect", "circle"}, context, "node shape must be rect or circle")
                require(entry.get("role") in ROLES and isinstance(entry.get("text"), str), context, "node role/text invalid")
                x, y, w, h = (entry.get(key) for key in ("x", "y", "width", "height"))
                require(all(type(v) is int for v in (x, y, w, h)) and x >= 0 and y >= 0
                        and w > 0 and h > 0 and x + w <= width and y + h <= height,
                        context, f"node {entry['id']} exceeds scene bounds")
                require(entry["shape"] != "circle" or w == h, context, "circle width and height must match")
            elif group == "labels":
                x, y = entry.get("x"), entry.get("y")
                require(type(x) is int and type(y) is int and 0 <= x <= width and 0 <= y <= height
                        and isinstance(entry.get("text"), str), context, "label position/text invalid")
    unique(ids, context + " scene")
    for edge in scene["edges"]:
        require(edge.get("from") in node_ids and edge.get("to") in node_ids, context,
                f"edge {edge['id']} references missing node")
        require(isinstance(edge.get("label"), str) and type(edge.get("directed")) is bool
                and edge.get("role") in ROLES, context, f"edge {edge['id']} fields invalid")


def validate_example(data, directory):
    context = str(directory / "example.json")
    require(isinstance(data, dict), context, "example must be an object")
    require(nonempty(data.get("id")) and nonempty(data.get("title")) and nonempty(data.get("invariant")),
            context, "id, title, and invariant are required")
    require(data["id"] == f"{directory.parent.name}-{directory.name}", context,
            "example ID must match its day and directory")
    variants = data.get("variants")
    require(isinstance(variants, list) and len(variants) == 4, context, "four languages are required")
    languages = [v.get("language") if isinstance(v, dict) else None for v in variants]
    require(set(languages) == LANGUAGES and len(set(languages)) == 4, context, "four unique languages are required")
    lengths = {}
    for variant in variants:
        lang = variant["language"]
        file = variant.get("file")
        require(isinstance(file, str) and file == Path(file).name and file not in {"", ".", ".."}
                and file in DEFAULT_FILES[lang], context, f"{lang} file must be safe and named one of {DEFAULT_FILES[lang]}")
        source = directory / file
        require(source.is_file() and not source.is_symlink(), context, f"{lang} source file missing or unsafe: {file}")
        length = len(source.read_text(encoding="utf-8").splitlines())
        start, end = variant.get("algorithmStart"), variant.get("algorithmEnd")
        require(type(start) is int and type(end) is int and 1 <= start <= end <= length,
                context, f"{lang} algorithm display bounds invalid")
        lengths[lang] = length
    cases = data.get("cases")
    require(isinstance(cases, list) and len(cases) >= 2, context, "normal and edge cases are required")
    require(all(isinstance(case, dict) for case in cases), context, "cases must be objects")
    unique([case.get("id") for case in cases], context + " cases")
    for case in cases:
        ctx = f"{context} case {case.get('id')}"
        require(nonempty(case.get("id")) and nonempty(case.get("label")), ctx, "case ID/label required")
        require(isinstance(case.get("input"), str) and case["input"].endswith("\n"), ctx, "input must end with newline")
        require(isinstance(case.get("expectedOutput"), str), ctx, "expectedOutput must be a string")
        frames = case.get("frames")
        require(isinstance(frames, list) and len(frames) >= 2, ctx, "initial and terminal frames required")
        for index, frame in enumerate(frames):
            frame_ctx = f"{ctx} frame {index}"
            require(isinstance(frame, dict) and nonempty(frame.get("event"))
                    and nonempty(frame.get("explanation")), frame_ctx, "event/explanation required")
            lines = frame.get("lines")
            require(isinstance(lines, dict) and set(lines) == LANGUAGES, frame_ctx, "lines need all four languages")
            for lang, numbers in lines.items():
                require(isinstance(numbers, list) and bool(numbers) and all(type(n) is int and 1 <= n <= lengths[lang]
                        for n in numbers), frame_ctx, f"{lang} lines out of source bounds")
            variables = frame.get("variables")
            require(isinstance(variables, list), frame_ctx, "variables must be a list")
            names = []
            for var in variables:
                require(isinstance(var, dict) and nonempty(var.get("name")) and isinstance(var.get("value"), str),
                        frame_ctx, "variable name/value invalid")
                names.append(var["name"])
            require(len(names) == len(set(names)), frame_ctx, "duplicate variable names")
            validate_scene(frame.get("scene"), frame_ctx)
    return variants, cases


def process(command, *, cwd=None, input_text=None, timeout=5, env=None):
    return subprocess.run(command, cwd=cwd, input=input_text, text=True, capture_output=True,
                          timeout=timeout, env=env, check=False)


def tools_for(lang, overrides):
    default = VERSION_COMMANDS[lang][0]
    return overrides.get(default, default)


def toolchain(lang, overrides):
    command = [tools_for(lang, overrides), *VERSION_COMMANDS[lang][1:]]
    try:
        result = process(command, timeout=5)
    except (OSError, subprocess.TimeoutExpired) as exc:
        return None, command, f"toolchain unavailable: {exc}"
    if result.returncode:
        return None, command, f"toolchain version failed: {result.stderr.strip()}"
    version = (result.stdout or result.stderr).strip().splitlines()[0]
    if lang == "python":
        match = re.search(r"Python (\d+)\.(\d+)", version)
        if not match or tuple(map(int, match.groups())) < (3, 11):
            return version, command, "Python 3.11+ required"
    if lang == "java":
        match = re.search(r"javac (\d+)", version)
        if not match or int(match.group(1)) < 17:
            return version, command, "Java 17+ required"
    return version, command, None


def compile_variant(lang, source, build_dir, overrides):
    env = os.environ.copy()
    if lang == "python":
        return [tools_for(lang, overrides), str(source)], None, None, None
    if lang == "cpp":
        executable = build_dir / "example"
        command = [tools_for(lang, overrides), "-std=c++17", "-O2", str(source), "-o", str(executable)]
        run_command = [str(executable)]
    elif lang == "java":
        command = [tools_for(lang, overrides), "--release", "17", "-d", str(build_dir), str(source)]
        run_command = [overrides.get("java", "java"), "-cp", str(build_dir), "Main"]
    else:
        executable = build_dir / "example"
        go_source = build_dir / "main.go" if source.name == "main.go.txt" else source
        command = [tools_for(lang, overrides), "build", "-o", str(executable), str(go_source)]
        run_command = [str(executable)]
        env.update(GOWORK="off", GO111MODULE="off", GOCACHE="/tmp/leetgrinder-go-cache")
    try:
        if lang == "go" and source.name == "main.go.txt":
            shutil.copyfile(source, go_source)
        result = process(command, cwd=build_dir, timeout=60, env=env)
    except (OSError, subprocess.TimeoutExpired) as exc:
        return run_command, command, str(exc), None
    if result.returncode:
        return run_command, command, f"compile exit {result.returncode}: {result.stderr.strip()}", result.stderr
    return run_command, command, None, result.stderr


def compare_output(stdout, case):
    errors = []
    lines = stdout.splitlines()
    expected_count = len(case["frames"]) + 1
    if len(lines) != expected_count:
        errors.append(f"record count {len(lines)}, expected {expected_count} (one event per frame plus result)")
    records = []
    for index, line in enumerate(lines):
        try:
            record = json.loads(line)
        except json.JSONDecodeError as exc:
            errors.append(f"record {index} invalid JSON: {exc.msg}")
            continue
        if not isinstance(record, dict):
            errors.append(f"record {index} must be a JSON object")
        records.append(record)
    for index, frame in enumerate(case["frames"]):
        if index >= len(records) or not isinstance(records[index], dict):
            continue
        expected = {"event": frame["event"], "variables": frame["variables"]}
        if records[index] != expected:
            for key in ("event", "variables"):
                if records[index].get(key) != expected[key]:
                    errors.append(f"frame {index} {key}: expected {expected[key]!r}, got {records[index].get(key)!r}")
            if set(records[index]) != set(expected):
                errors.append(f"frame {index} event record has unexpected/missing fields")
    final_index = len(case["frames"])
    if final_index < len(records) and records[final_index] != {"result": case["expectedOutput"]}:
        errors.append(f"result: expected {case['expectedOutput']!r}, got {records[final_index]!r}")
    return errors


def verify_example(directory, *, run_timeout=5, tool_commands=None):
    directory = Path(directory)
    result = {"example": directory.name, "day": directory.parent.name, "passed": False, "errors": [], "runs": [], "toolchains": {}}
    tool_commands = tool_commands or {}
    try:
        require(not directory.is_symlink() and not (directory / "example.json").is_symlink(),
                str(directory), "unsafe symlinked example directory or metadata")
        directory = directory.resolve()
        data = json.loads((directory / "example.json").read_text(encoding="utf-8"))
        variants, cases = validate_example(data, directory)
        result["example"] = data["id"]
    except (OSError, UnicodeError, json.JSONDecodeError, InvalidExample, TypeError) as exc:
        result["errors"].append(str(exc))
        return result
    with tempfile.TemporaryDirectory(prefix="leetgrinder-verify-") as temporary:
        for variant in variants:
            lang = variant["language"]
            version, version_command, error = toolchain(lang, tool_commands)
            result["toolchains"][lang] = {"version": version, "command": version_command}
            if error:
                result["errors"].append(f"{directory.name} {lang}: {error}")
                for case in cases:
                    result["runs"].append({"case": case["id"], "language": lang, "passed": False,
                                           "errors": [error], "command": [], "compileCommand": [], "toolchainVersion": version})
                continue
            build_dir = Path(temporary) / lang
            build_dir.mkdir()
            command, compile_command, compile_error, diagnostics = compile_variant(lang, directory / variant["file"], build_dir, tool_commands)
            if compile_error:
                result["errors"].append(f"{directory.name} {lang}: {compile_error}")
            for case in cases:
                run = {"case": case["id"], "language": lang, "passed": False, "errors": [],
                       "command": command, "compileCommand": compile_command or [],
                       "toolchainVersion": version, "compileDiagnostics": diagnostics or "", "stdout": "", "stderr": ""}
                result["runs"].append(run)
                if compile_error:
                    run["errors"].append(compile_error)
                    continue
                env = os.environ.copy()
                if lang == "go":
                    env.update(GOWORK="off", GO111MODULE="off", GOCACHE="/tmp/leetgrinder-go-cache")
                try:
                    completed = process(command, cwd=build_dir, input_text=case["input"], timeout=run_timeout, env=env)
                except subprocess.TimeoutExpired as exc:
                    run["errors"].append(f"execution timeout after {run_timeout}s")
                    run["stdout"] = exc.stdout.decode(errors="replace") if isinstance(exc.stdout, bytes) else exc.stdout or ""
                    run["stderr"] = exc.stderr.decode(errors="replace") if isinstance(exc.stderr, bytes) else exc.stderr or ""
                    continue
                except OSError as exc:
                    run["errors"].append(f"execution failed: {exc}")
                    continue
                run["stdout"], run["stderr"] = completed.stdout, completed.stderr
                if completed.returncode:
                    run["errors"].append(f"execution exit {completed.returncode}: {completed.stderr.strip()}")
                run["errors"].extend(compare_output(completed.stdout, case))
                run["passed"] = not run["errors"]
    result["passed"] = not result["errors"] and all(run["passed"] for run in result["runs"])
    return result


def verify_days(root, days):
    root = Path(root)
    report = {"passed": False, "errors": [], "examples": []}
    for day in days:
        day_dir = root / f"day-{day:02d}"
        directories = sorted(path for path in day_dir.iterdir() if path.is_dir()) if day_dir.is_dir() else []
        if not directories:
            report["errors"].append(f"day-{day:02d}: no examples found")
        for directory in directories:
            if not (directory / "example.json").is_file():
                report["errors"].append(f"day-{day:02d} {directory.name}: missing example.json")
                continue
            report["examples"].append(verify_example(directory))
    report["passed"] = not report["errors"] and bool(report["examples"]) and all(x["passed"] for x in report["examples"])
    return report


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    selection = parser.add_mutually_exclusive_group(required=True)
    selection.add_argument("--days", help="comma-separated day numbers")
    selection.add_argument("--all", action="store_true", help="verify all 84 course days")
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[2] / "internal" / "leetgrinder" / "examples")
    parser.add_argument("--report", type=Path)
    args = parser.parse_args(argv)
    if args.all:
        days = list(range(1, 85))
    else:
        try:
            days = [int(part) for part in args.days.split(",")]
        except ValueError:
            parser.error("--days must contain comma-separated integers")
        if not days or any(day < 1 or day > 84 for day in days) or len(days) != len(set(days)):
            parser.error("--days must contain unique integers from 1 to 84")
    report = verify_days(args.root, days)
    if args.report:
        args.report.parent.mkdir(parents=True, exist_ok=True)
        args.report.write_text(json.dumps(report, indent=2) + "\n", encoding="utf-8")
    for error in report["errors"]:
        print(error, file=sys.stderr)
    for example in report["examples"]:
        for error in example["errors"]:
            print(error, file=sys.stderr)
        for run in example["runs"]:
            for error in run["errors"]:
                print(f"{example['day']} {example['example']} case {run['case']} {run['language']}: {error}", file=sys.stderr)
    return 0 if report["passed"] else 1


if __name__ == "__main__":
    raise SystemExit(main())
