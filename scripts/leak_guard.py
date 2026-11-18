#!/usr/bin/env python3
"""Fail when text in this public repo names something internal.

The banned names are not in this repo: only the SHA-256 of each (lowercase)
is, in scripts/banned-terms.sha256, so the list itself can't leak them.
The check splits text into tokens, lowercases them, hashes it and every run of
its dot- and hyphen-separated parts (so "x.internal.example" matches
"internal.example" and "name-of-thing-v2" matches "name-of-thing") and reports file:line for every hit, never the term.

  python3 scripts/leak_guard.py --diff BASE HEAD   # lines added between two commits + their messages
  python3 scripts/leak_guard.py --all              # every tracked file
  python3 scripts/leak_guard.py --hash TERM        # print the hash to add a new term

Exits 1 on a hit, 0 otherwise. Standard library only.
"""

import hashlib
import pathlib
import re
import subprocess
import sys

HASHES_FILE = pathlib.Path(__file__).with_name("banned-terms.sha256")

# A token is a run of letters, digits, dots, hyphens and underscores, so
# "https://a.example/b-c#1" gives "a.example" and "b-c".
TOKEN = re.compile(r"[a-z0-9._-]+")


def digest(term: str) -> str:
    return hashlib.sha256(term.strip().lower().encode()).hexdigest()


def load_hashes() -> set[str]:
    hashes = set()
    for line in HASHES_FILE.read_text().splitlines():
        line = line.split("#", 1)[0].strip()
        if line:
            hashes.add(line)
    return hashes


def runs(token: str, sep: str):
    """Every run of consecutive sep-separated parts: "a.b.c" gives a, a.b, a.b.c, b, b.c, c."""
    parts = token.split(sep)
    for i in range(len(parts)):
        for j in range(i + 1, len(parts) + 1):
            yield sep.join(parts[i:j])


def candidates(token: str):
    for by_dot in runs(token.strip("._-"), "."):
        for by_hyphen in runs(by_dot, "-"):
            c = by_hyphen.strip("._-")
            if c:
                yield c


def hits(text: str, hashes: set[str]) -> bool:
    for token in TOKEN.findall(text.lower()):
        for c in candidates(token):
            if digest(c) in hashes:
                return True
    return False


def git(*args: str) -> str:
    return subprocess.run(["git", *args], check=True, capture_output=True, text=True).stdout


def scan_all(hashes: set[str]) -> list[str]:
    found = []
    for path in git("ls-files", "-z").split("\0"):
        if not path or path == "scripts/banned-terms.sha256":
            continue
        if hits(path, hashes):
            found.append(f"{path}: the file path")
        try:
            text = pathlib.Path(path).read_text(errors="ignore")
        except (IsADirectoryError, FileNotFoundError):
            continue
        for n, line in enumerate(text.splitlines(), 1):
            if hits(line, hashes):
                found.append(f"{path}:{n}")
    return found


def scan_diff(base: str, head: str, hashes: set[str]) -> list[str]:
    found = []
    path, line_no = None, 0
    diff = git("diff", "--no-color", "--unified=0", f"{base}...{head}")
    for line in diff.splitlines():
        if line.startswith("+++ "):
            path = line[6:] if line.startswith("+++ b/") else None
            if path and hits(path, hashes):
                found.append(f"{path}: the file path")
        elif line.startswith("@@"):
            m = re.search(r"\+(\d+)", line)
            line_no = int(m.group(1)) if m else 0
        elif line.startswith("+") and path:
            if path != "scripts/banned-terms.sha256" and hits(line[1:], hashes):
                found.append(f"{path}:{line_no}")
            line_no += 1
    for sha in git("rev-list", f"{base}..{head}").split():
        if hits(git("log", "-1", "--format=%B", sha), hashes):
            found.append(f"commit {sha[:12]}: the commit message")
        if hits(git("log", "-1", "--format=%an %ae %cn %ce", sha), hashes):
            found.append(f"commit {sha[:12]}: the author or committer")
    return found


def main(argv: list[str]) -> int:
    if len(argv) == 2 and argv[0] == "--hash":
        print(digest(argv[1]))
        return 0
    hashes = load_hashes()
    if argv == ["--all"]:
        found = scan_all(hashes)
    elif len(argv) == 3 and argv[0] == "--diff":
        found = scan_diff(argv[1], argv[2], hashes)
    else:
        print(__doc__, file=sys.stderr)
        return 2
    for where in found:
        # GitHub annotation when the place is a file line; the term is never printed.
        if ":" in where and not where.startswith("commit "):
            file, _, rest = where.partition(":")
            print(f"::error file={file},line={rest if rest.isdigit() else 1}::names something internal (a banned term). Remove it.")
        print(f"banned term at {where}")
    if found:
        print(f"{len(found)} place(s) name something internal. This repo is public: remove them (and rewrite the commits if it's in a message).", file=sys.stderr)
        return 1
    print(f"no banned terms ({len(hashes)} hashed terms checked)")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
