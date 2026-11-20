## What this changes

<!-- One plugin or one fix. Which provider, which source, what kind of change it reports. -->

## Checklist

- [ ] **Metadata complete in the registration** (`plugin.RegisterProvider`: provider id, name, category, homepage; each source's id, kind, layer, what it watches, URLs, packages, page), and I read the words. There is no list to update. Not applicable: <!-- say why -->
- [ ] Every commit is signed off (`git commit -s`, DCO)
- [ ] Golden test over a real saved copy of the source (`testdata/golden/<api>.json`)
- [ ] `testdata/README.md` lists each file: URL, date fetched, license, quote of the part the plugin reads
- [ ] One pinned drop test: a realistic input that produces no event
- [ ] `providers/<api>.md` written or updated from `providers/_template.md`, including **Why this source**
- [ ] Fetches only through `env.Get` / `core/kit/`
- [ ] `make check` passes locally (`make plugins` prints the source as you expect)
- [ ] Nothing private: no credentials, internal hosts, private links, customer names

## Notes for the reviewer

<!-- Anything you weren't sure about: borderline changes you chose not to emit, trimmed testdata, golden diffs. -->
