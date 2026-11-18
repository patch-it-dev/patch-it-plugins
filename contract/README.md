# Contract fixtures

`fixtures/` is a copy of the change-event fixtures in [patch-it's `fixtures/`](https://github.com/patch-it-dev/patch-it/tree/main/fixtures): the change-event contract is owned by patch-it ([docs/contract](https://github.com/patch-it-dev/patch-it/tree/main/docs/contract)), and `event/` is its Go binding. `event/` and `migrate/` test against these files, so the plugins and the patch-it client validate the same events.

- **Don't edit them here.** Change the contract in patch-it first, then copy its `fixtures/change-events*.json` over these.
- The top-level `_comment` of a few files is worded differently here (where the fixture came from); the events themselves are identical.
- `CI_contract` checks the copy on every PR and weekly: `python3 scripts/check_contract.py <path to patch-it>/fixtures`.
