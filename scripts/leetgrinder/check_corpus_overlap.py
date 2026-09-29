#!/usr/bin/env python3
"""Flag long exact prose overlaps with the local research corpus; never embed it."""
import argparse
import json
import re
from pathlib import Path


def words(value: str) -> list[str]:
    return re.findall(r"[a-z0-9]+", value.lower())


def strings(value):
    if isinstance(value, str):
        yield value
    elif isinstance(value, list):
        for item in value:
            yield from strings(item)
    elif isinstance(value, dict):
        for item in value.values():
            yield from strings(item)


def code_lines(value: str) -> list[str]:
    return [" ".join(line.split()) for line in value.splitlines() if line.strip()]


def find_code_overlaps(root: Path, corpus: Path, width: int = 5) -> list[tuple[Path, Path]]:
    fingerprints = {}
    for article in corpus.rglob("*.md"):
        for block in re.findall(r"```[^\n]*\n(.*?)\n```", article.read_text(errors="replace"), re.DOTALL):
            lines = code_lines(block)
            for index in range(len(lines) - width + 1):
                fingerprints.setdefault(tuple(lines[index:index + width]), article)

    examples = root / "internal/leetgrinder/examples"
    matches = set()
    for file in examples.glob("day-*/*/*"):
        if not file.is_file() or file.name not in {"main.cpp", "main.py", "Main.java", "main.go.txt"}:
            continue
        lines = code_lines(file.read_text())
        for index in range(len(lines) - width + 1):
            article = fingerprints.get(tuple(lines[index:index + width]))
            if article:
                matches.add((file.relative_to(root), article.relative_to(corpus)))
    return sorted(matches)


def find_overlaps(root: Path, corpus: Path, width: int = 15) -> list[tuple[Path, Path]]:
    fingerprints = {}
    for article in corpus.rglob("*.md"):
        tokens = words(article.read_text(errors="replace"))
        for index in range(len(tokens) - width + 1):
            fingerprints.setdefault(tuple(tokens[index:index + width]), article)

    authored = root / "internal/leetgrinder"
    files = list((authored / "lessons").glob("day-*.json"))
    files += list((authored / "examples").glob("day-*/*/example.json"))
    matches = set()
    for file in files:
        for value in strings(json.loads(file.read_text())):
            tokens = words(value)
            for index in range(len(tokens) - width + 1):
                article = fingerprints.get(tuple(tokens[index:index + width]))
                if article:
                    matches.add((file.relative_to(root), article.relative_to(corpus)))
    return sorted(matches)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--root", type=Path, default=Path(__file__).resolve().parents[2])
    parser.add_argument("--corpus", type=Path, default=Path.home() / "Documents/data/algomonster/articles")
    args = parser.parse_args()
    if not args.corpus.is_dir():
        parser.error(f"research corpus unavailable: {args.corpus}")
    matches = find_overlaps(args.root, args.corpus)
    for authored, article in matches:
        print(f"{authored}: 15-word exact overlap with {article}")
    print(f"{len(matches)} authored/source file pairs with long exact prose overlap")
    code_matches = find_code_overlaps(args.root, args.corpus)
    for authored, article in code_matches:
        print(f"{authored}: five-line exact code overlap with {article}")
    print(f"{len(code_matches)} program/source file pairs with five-line exact code overlap")
    raise SystemExit(bool(matches or code_matches))


if __name__ == "__main__":
    main()
