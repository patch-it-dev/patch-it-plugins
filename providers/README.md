# Providers

What patch-it reads: one folder per provider, `providers/<provider>/`, holding everything about that provider and nothing about any other.

- `provider.go`: the provider's registration (`plugin.RegisterProvider`): its id, name, category and homepage, and every source of it, whatever its kind, with the source's layer, what it watches, the URLs it reads and the packages it covers. Sources are built from the kinds' exported types (`kinds/<kind>`, how to read).
- `README.md`: the provider's page, one section per source: what it watches, **why this source** (the link, the date it was checked, why it's the best source, the alternatives not used, a short quote), how the provider announces changes, what the source reads and ignores, and its known gaps. Start from [`docs/provider-template.md`](../docs/provider-template.md).
- `<source>.go`: only when a source needs code of its own: an OpenAPI adapter (where the spec is, how its versions are listed, what the differ keeps), a feed's reader.
- `testdata/`: the saved copies the tests replay, each with a row in `testdata/README.md` (URL, date fetched, license, quote), and the golden files, `testdata/golden/`, with every event the tests must store.
- `*_test.go`: each source's golden test and its pinned drop test.

[`all/`](all/) imports every provider, so a program that imports it runs every source; its tests fail when a provider's folder isn't imported, a source's metadata or page is missing, or a package imports what it shouldn't (a kind imports no provider; a provider imports only `core/` and `kinds/`).

There's no list of providers or sources here: `go run ./cmd/list-plugins` prints them from the registry, and each release carries that list as `plugins.json`.
