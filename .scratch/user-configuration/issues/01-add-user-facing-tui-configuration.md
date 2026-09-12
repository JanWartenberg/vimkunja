# 01 — Add user-facing TUI configuration

**What to build:** Users can configure supported Vimkunja behavior, starting with timing preferences already exposed programmatically.

**Implementation prerequisites:** Needs a configuration source, precedence, and compatibility specification.

**Category:** enhancement

**Priority:** Low

**Status:** implemented

## Decisions confirmed

- Scope is limited to local TUI behavior. Workspace/data preferences belong in a separate ticket covering default Project, default View, startup filters/search scope, and other persisted navigation/workspace preferences. Database selection (`--db`) remains a separate operational concern.
- The persistent configuration file uses TOML at the platform-standard user config directory, under `vimkunja/config.toml` (via `os.UserConfigDir()`).
- Settings are grouped under `[tui]`.
- Precedence is CLI flags > environment variables > TOML config > built-in defaults.
- The first implementation supports TOML and CLI flags. Environment-variable overrides are deferred.
- Initial user-facing settings are `leader_timeout` and `ctrl_w_timeout`; cursor-shape configuration is deferred.
- CLI overrides are named `--leader-timeout` and `--ctrl-w-timeout`; they accept the same Go duration syntax as TOML. Invalid explicit CLI values prevent startup with a non-zero exit status.
- Timeout values use Go duration syntax (for example, `"500ms"` and `"2s"`) and must be positive; zero and negative values are invalid.
- A missing config file silently uses defaults and is not auto-created.
- Invalid TOML/values, unreadable existing config files, and unknown settings/sections fall back to defaults without preventing startup. They produce warnings in the startup footer and `:errors`, and should also be reported on the command line where possible, with the most specific file/setting/value diagnostic possible. Unambiguous supported settings are still parsed and applied; unknown sections are ignored.
- Invalid explicit CLI values (malformed duration, zero, or negative) prevent startup with a direct command-line error and non-zero exit status.
- Documentation must include clear examples for every supported setting: a verbose reference and complete TOML example in `README.md`, plus concise flag descriptions/examples in `--help`. No separate config document is needed.
- Config warnings are printed to stderr before the TUI starts; the footer shows a concise warning and `:errors` retains all details. Multiple warnings use a compact footer summary while stderr and `:errors` list every warning. Recoverable config warnings do not affect the eventual process exit status. The tutorial also loads the user TUI configuration.
- Valid settings are applied when another setting has an invalid value; only the invalid field falls back to its default. Malformed TOML uses defaults for the whole file.
- A custom `--config` path is deferred; the platform-standard location is used initially.
- Configuration is read at startup only; changes take effect on the next launch.
- Tutorial support is optional and should be added only if shared wiring makes it cheap; it must not expand the main design.
- Workspace/data configuration is explicitly deferred until this local TUI configuration work is complete and will be triaged separately in `.scratch/workspace-configuration/issues/01-specify-workspace-preferences-configuration.md`.

## Open decisions

- Environment-variable naming and support remain deferred.
- Whether and how to provide a future config example/template command.

## Implementation

- Added TOML loading with positive-duration validation and field-level fallback.
- Added `--leader-timeout` and `--ctrl-w-timeout` to both entrypoints.
- Added startup warning reporting to stderr, the TUI footer, and the session error log.
- Added configuration documentation and tests.

## Further context

Archived source: `SPEC_FUNDAMENT.md`, user-facing configuration deferred; current `tui.Config` is programmatic only.
