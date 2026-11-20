# Saved releases

The golden file, `golden/openai-1.1.2.json`, is what the kind stores from the 1.1.1 → 1.1.2 pair (`go test ./plugins/apisurface -update`).

| File | URL | Saved | License | Quote |
|---|---|---|---|---|
| `openai-1.1.1-py3-none-any.trimmed.whl` | https://files.pythonhosted.org/packages/f7/86/ea99e3f966a639a3256e2f79d78cd63d45d5eb651a5457ccbdcfd9eed24a/openai-1.1.1-py3-none-any.whl | 2026-11-20 | Apache-2.0 (openai-python; the wheel's `licenses/LICENSE` is kept in it). Trimmed: only `openai/__init__.py`, `openai/types/__init__.py`, every file under `openai/types/beta/`, and the dist-info `METADATA`, `WHEEL` and `LICENSE`, each copied byte for byte (46 of the wheel's files); the plugin reads the same symbols under `openai.types.beta` from it as from the whole wheel. | `class AsssitantDeleted(BaseModel):` (`openai/types/beta/asssitant_deleted.py`, line 10) |
| `openai-1.1.2-py3-none-any.trimmed.whl` | https://files.pythonhosted.org/packages/dd/82/b92f73453ea318c0d46f31aeb56e9d94a42606c010fb72a513f4a3cd4bac/openai-1.1.2-py3-none-any.whl | 2026-11-20 | The same, trimmed the same way. | `from .assistant_deleted import AssistantDeleted as AssistantDeleted` (`openai/types/beta/__init__.py`) |
| `pypi-openai-excerpt.json` | https://pypi.org/pypi/openai/json | 2026-11-20 | PyPI metadata. Trimmed: `releases` to 1.0.0rc1 (a pre-release, left out) and 1.1.0 to 1.1.2, each file to `filename`, `packagetype`, `upload_time_iso_8601`, `yanked` | |
| `pypi-openai-1.1.1.json`, `pypi-openai-1.1.2.json` | https://pypi.org/pypi/openai/1.1.1/json, …/1.1.2/json | 2026-11-20 | PyPI metadata. Trimmed: `info.name`, `info.version`, and `urls` to `filename`, `packagetype`, `url`, `yanked` | |
