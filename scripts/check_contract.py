#!/usr/bin/env python3
"""Check contract/fixtures/ against patch-it's fixtures/.

patch-it (https://github.com/patch-it-dev/patch-it) owns the change-event
contract; this repo carries a copy of its change-event fixtures so core/event/
tests against the same events the patch-it client does. The copies
must hold the same JSON as patch-it's. Only the top-level "_comment" fields may
differ: they describe where a fixture came from, and this repo words them for
a public audience.

  python3 scripts/check_contract.py UPSTREAM_FIXTURES_DIR

Exits 1 when a fixture differs or is missing on either side, 0 otherwise.
Standard library only.
"""

import json
import pathlib
import sys

OURS = pathlib.Path(__file__).resolve().parent.parent / "contract" / "fixtures"


def load(path: pathlib.Path):
    data = json.loads(path.read_text(encoding="utf-8"))
    if isinstance(data, dict):
        data.pop("_comment", None)
    return data


def main() -> int:
    if len(sys.argv) != 2:
        print(__doc__, file=sys.stderr)
        return 2
    upstream = pathlib.Path(sys.argv[1])
    theirs = {p.name: p for p in upstream.glob("change-events*.json")}
    ours = {p.name: p for p in OURS.glob("*.json")}
    if not theirs:
        print(f"::error::no change-events*.json in {upstream}")
        return 1
    status = 0
    for name, path in sorted(theirs.items()):
        if name not in ours:
            print(f"::error file=contract/fixtures/{name}::missing: patch-it's fixtures/{name} has no copy here")
            status = 1
        elif load(path) != load(ours[name]):
            print(f"::error file=contract/fixtures/{name}::differs from patch-it's fixtures/{name} (beyond _comment). Copy it again.")
            status = 1
    for name in sorted(set(ours) - set(theirs)):
        print(f"::error file=contract/fixtures/{name}::not in patch-it's fixtures/")
        status = 1
    if status == 0:
        print(f"contract/fixtures matches patch-it's fixtures/ ({len(theirs)} files)")
    return status


if __name__ == "__main__":
    sys.exit(main())
