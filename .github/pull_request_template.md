## What this changes

<!-- One plugin or one fix. Which provider, which source, what kind of change it reports. -->

## Checklist

- [ ] **Added to / updated in `docs/plugins.md`** (new plugin: a row; changed id, provider, source, what it watches, format or status: the row updated; removed: the row removed). Not applicable: <!-- say why -->
- [ ] Every commit is signed off (`git commit -s`, DCO)
- [ ] Golden test over a real saved copy of the source (`testdata/golden/<api>.json`)
- [ ] `testdata/README.md` lists each file: URL, date fetched, license, quote of the part the plugin reads
- [ ] One pinned drop test: a realistic input that produces no event
- [ ] `providers/<api>.md` written or updated from `providers/_template.md`, including **Why this source**
- [ ] Fetches only through `env.Get` / `kit/`
- [ ] `make check` passes locally (and `make plugins-json` was run if a plugin or its row changed)
- [ ] Nothing private: no credentials, internal hosts, private links, customer names

## Notes for the reviewer

<!-- Anything you weren't sure about: borderline changes you chose not to emit, trimmed testdata, golden diffs. -->
