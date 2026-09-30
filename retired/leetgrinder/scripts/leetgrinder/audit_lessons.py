#!/usr/bin/env python3
"""Check authored lesson coverage without using the private research corpus."""
import argparse
import json
import re
from pathlib import Path

LANGUAGES = {"cpp", "python", "java", "go"}
HEADINGS = ("intuition", "baseline", "worked example", "complexity", "before you finish")
EXAMPLE_FILES = {"example.json", "main.cpp", "main.py", "Main.java", "main.go.txt"}


def embedded_asset_errors(root: Path) -> list[str]:
    """Reject stray files that Go's directory embed would include in the site binary."""
    errors = []
    lessons = root / "internal/leetgrinder/lessons"
    examples = root / "internal/leetgrinder/examples"
    for path in lessons.rglob("*"):
        if path.is_file() and (path.parent != lessons or not re.fullmatch(r"day-\d{2}\.json", path.name)):
            errors.append(f"unexpected embedded lesson asset: {path.relative_to(root)}")
    for path in examples.rglob("*"):
        if not path.is_file():
            continue
        parts = path.relative_to(examples).parts
        if len(parts) != 3 or not re.fullmatch(r"day-\d{2}", parts[0]) or parts[2] not in EXAMPLE_FILES:
            errors.append(f"unexpected embedded example asset: {path.relative_to(root)}")
    return sorted(errors)


def audit(root: Path, days=range(1, 85)) -> dict:
    entries = []
    for day in days:
        key = f"day-{day:02d}"
        entry = {"day": day, "examples": 0, "cases": 0, "languageRuns": 0, "errors": []}
        lesson_path = root / "internal/leetgrinder/lessons" / f"{key}.json"
        if not lesson_path.is_file():
            entry["errors"].append(f"day {day:02d} lesson missing")
            entries.append(entry)
            continue
        try:
            lesson = json.loads(lesson_path.read_text())
        except (OSError, ValueError) as exc:
            entry["errors"].append(f"invalid lesson JSON: {exc}")
            entries.append(entry)
            continue
        if lesson.get("day") != day:
            entry["errors"].append("day number does not match filename")
        if not lesson.get("objectives"):
            entry["errors"].append("learning objectives missing")
        sections = lesson.get("sections") or []
        headings = " ".join(s.get("heading", "").lower() for s in sections)
        for heading in HEADINGS:
            if heading not in headings:
                entry["errors"].append(f"section for {heading} missing")
        if not any(s.get("diagram") for s in sections):
            entry["errors"].append("static concept diagram missing")
        if not any(s.get("checks") for s in sections):
            entry["errors"].append("knowledge checks missing")
        example_ids = [example for section in sections for example in section.get("exampleIds", [])]
        if len(example_ids) != len(set(example_ids)):
            entry["errors"].append("duplicate example ID")
        if not example_ids:
            entry["errors"].append("runnable example missing")
        for example_id in example_ids:
            prefix = key + "-"
            if not example_id.startswith(prefix) or not example_id[len(prefix):]:
                entry["errors"].append(f"invalid example ID {example_id}")
                continue
            directory = root / "internal/leetgrinder/examples" / key / example_id[len(prefix):]
            metadata = directory / "example.json"
            if not metadata.is_file():
                entry["errors"].append(f"{example_id}: example metadata missing")
                continue
            try:
                example = json.loads(metadata.read_text())
            except (OSError, ValueError) as exc:
                entry["errors"].append(f"{example_id}: invalid JSON: {exc}")
                continue
            entry["examples"] += 1
            if example.get("id") != example_id:
                entry["errors"].append(f"{example_id}: ID mismatch")
            variants = example.get("variants") or []
            languages = {variant.get("language") for variant in variants}
            if languages != LANGUAGES or len(variants) != 4:
                entry["errors"].append(f"{example_id}: four languages required")
            for variant in variants:
                name = variant.get("file", "")
                if not name or Path(name).name != name or not (directory / name).is_file():
                    entry["errors"].append(f"{example_id}: source missing for {variant.get('language')}")
            cases = example.get("cases") or []
            if len(cases) < 2 or not any("edge" in case.get("id", "").lower() or case.get("id") in ("empty", "single", "duplicate", "repeated", "reset", "disjoint") for case in cases):
                entry["errors"].append(f"{example_id}: normal and edge cases required")
            for case in cases:
                if len(case.get("frames") or []) < 2:
                    entry["errors"].append(f"{example_id}/{case.get('id')}: at least two frames required")
            entry["cases"] += len(cases)
            entry["languageRuns"] += len(cases) * len(languages & LANGUAGES)
        entries.append(entry)
    asset_errors = embedded_asset_errors(root)
    return {
        "passed": not asset_errors and all(not entry["errors"] for entry in entries),
        "embeddedAssetErrors": asset_errors,
        "totals": {"days": len(entries), "completeDays": sum(not entry["errors"] for entry in entries), "examples": sum(entry["examples"] for entry in entries), "cases": sum(entry["cases"] for entry in entries), "languageRuns": sum(entry["languageRuns"] for entry in entries)},
        "days": entries,
    }


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[2])
    parser.add_argument("--days", help="comma-separated lesson numbers; default all 84")
    parser.add_argument("--report", type=Path, help="write full JSON report")
    args = parser.parse_args()
    days = [int(value) for value in args.days.split(",")] if args.days else range(1, 85)
    report = audit(args.root, days)
    if args.report:
        args.report.write_text(json.dumps(report, indent=2) + "\n")
    print(json.dumps(report["totals"]))
    for entry in report["days"]:
        for error in entry["errors"]:
            print(f"day {entry['day']:02d}: {error}")
    for error in report["embeddedAssetErrors"]:
        print(error)
    raise SystemExit(0 if report["passed"] else 1)


if __name__ == "__main__":
    main()
