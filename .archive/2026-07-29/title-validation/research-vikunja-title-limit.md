# Vikunja Task Title Limit Research

**Question:** Does Vikunja impose a maximum Task title length, particularly in the SQLite schema?

## Findings

1. The local Vikunja SQLite database stores `tasks.title` as `TEXT NOT NULL`; its `CREATE TABLE` statement has neither a `CHECK` constraint nor a length declaration. SQLite itself therefore does not impose a title-length limit here.
2. Vikunja's upstream `Task` model declares `Title` as `xorm:"TEXT not null"` with `valid:"minstringlength(1)"` and `minLength:"1"`; it declares no maximum length.
3. Upstream task creation rejects only an empty title (`t.Title == ""`).
4. Vimkunja currently likewise validates only that a title is non-empty.

## Conclusion

For the inspected local schema and current upstream Vikunja source, the compatible rule is **a non-empty title with no artificial maximum**. Any future title limit in Vimkunja would be a product-specific restriction, not a SQLite-schema or current-upstream compatibility requirement.

## Primary sources

- Local database, `internal/db/vikunja.db`, read immutable on 2026-07-29: `tasks.title TEXT NOT NULL` in the `tasks` table definition (schema version 126).
- Vikunja upstream [`pkg/models/tasks.go`](https://github.com/go-vikunja/api/blob/main/pkg/models/tasks.go): `Task.Title` declaration and `createTask` empty-title validation.
- Vimkunja [`internal/vimkunja/sqlite_store.go`](../../../internal/vimkunja/sqlite_store.go): `validateDraft` requires a title but sets no length limit.
