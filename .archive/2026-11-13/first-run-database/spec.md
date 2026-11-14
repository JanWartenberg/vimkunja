# First-run database

Accepted by Jan on 2026-11-13; implementation authorized. This specifies fresh
startup only, not general Project creation or database migrations.

## Startup contract

- Explicit `--db` selects an existing file; missing or empty explicit paths fail.
- Otherwise prefer the existing working-directory-relative `internal/db/vikunja.db`,
  including invalid/inaccessible paths (report the error; never silently switch).
- Without a legacy file, use `%LOCALAPPDATA%/vimkunja/vikunja.db` on Windows,
  `~/Library/Application Support/vimkunja/vikunja.db` on macOS, and
  `$XDG_DATA_HOME/vimkunja/vikunja.db` or `~/.local/share/vimkunja/vikunja.db` on Unix.
  Configured data roots must be absolute. Report the selected absolute path.
- Only an absent default file is initialized. Existing databases are never replaced,
  repaired or migrated. Compatibility checks remain unchanged.
- Install the embedded verified Vikunja 2.7.0 template with Local User (`local`),
  Inbox, four standard Views and three Kanban Buckets. No demo Tasks. The User has
  no usable password or email; using VOA requires its CLI password reset first.
- New directories use mode 0700 and the database uses 0600 where Unix permissions
  apply. Write, sync, close and validate a sibling temporary file, then publish with
  an atomic hard link that cannot overwrite an existing file. A concurrent loser
  opens the winner normally. Unsupported filesystems fail with a useful error.
- Failed preparation removes its temporary file; a crash may leave a harmless
  hidden temporary file. Existing orphaned WAL/SHM/journal files block creation.
- Keep tutorials independent and disposable. Do not change dependencies, the
  upstream schema, or any local user database.

## Template provenance and generation

The official v2.7.0 Linux amd64 executable reports v2.7.0; SHA256:
`9745ad79a0f6836a59e12357f2afa6de3d94a3ae2dd011d222d258fb6b6fc0f1`.
The full release ZIP SHA256 was previously verified against the release asset:
`569861fd690f9ffc812e39e20be3943b882d4549a19e556e47066e71e29ca1b9`.
Source commit: `a16be96aa454671fdf213b0fbe411dd38a098418`.
Upstream Vikunja is AGPL-3.0; attribution is in THIRD_PARTY_NOTICES.txt. This
generated database includes no server executable or implementation source.

Generate only from a fresh official server in an isolated temporary directory:

```bash
VIMKUNJA_VOA_BINARY=/absolute/path/to/vikunja-v2.7.0-linux-amd64 VIMKUNJA_STARTER_OUTPUT=/absolute/path/to/internal/vimkunja/starter/vikunja-2.7.0.sqlite.gz go test -tags=integration ./internal/vimkunja -run '^TestGenerateStarterDatabase$' -v -count=1
```

The generator registers a synthetic User using a random temporary password;
VOA creates Inbox and standard Views. It removes credentials, sessions, runtime
license status, and the optional saved-filter Project/Views/Buckets, normalizes
starter timestamps, checkpoints, and VACUUMs before compression. No real data or
machine-specific paths are used. Regeneration is logically reproducible; SQLite
layout and migration metadata may change the byte hash. Review and verify every
regenerated artifact. Shipped uncompressed SQLite SHA256:
`dfb08728212fa23dcfdd35cc21ae067929bf3d4908e7b2641ea5c23aaabaaf37`.

Verify with the real original application, using disposable installed copies:

```bash
VIMKUNJA_VOA_BINARY=/absolute/path/to/vikunja-v2.7.0-linux-amd64 go test -tags=integration ./internal/vimkunja -run '^TestStarterDatabaseOriginalAppRoundtrip$' -v -count=1
```

This sets the local User password with VOA's CLI, logs in, verifies Inbox/Views,
reads a Vimkunja-created Task, edits it, creates a subsequent Task, then closes
VOA and checks Vimkunja can reopen and read its changes. Access is sequential.
