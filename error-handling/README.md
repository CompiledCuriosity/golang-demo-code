# Go error handling: %v, %w, the error chain, errors.Is and errors.As

Every program here is one that produced an output shown in the video. The running example is always the same: `findUser` fails with the sentinel `ErrNotFound`, `loadProfile` adds context (`profile 42: ...`), and the caller decides between 404 and 500. Each program is one `main.go`.

Run any of them from the repo root:

```
go run ./error-handling/hook
```

Go 1.25 or later builds everything except `astype` and `stdlib`, which use `errors.AsType` (added in Go 1.26) and carry a `//go:build go1.26` line, so an older toolchain skips them instead of failing the build. The outputs in the video were measured with go1.27.1.

## The programs, in the order the video walks them

| Folder | What it shows |
| --- | --- |
| `hook` | The bug. `loadProfile` wraps with `%v`, the caller checks with `errors.Is`, and a missing user gets `500 profile 42: not found`. |
| `unwrapped` | Before anyone added context: `loadProfile` returns the error it got, `err == ErrNotFound` is true, `404 not found`. |
| `wrapv` | Context added with `%v`, still checked with `==`: `500`. The `%v` error is a brand new error with the old message copied into it. |
| `strmatch` | The shortcut: `strings.Contains(err.Error(), "not found")` is true for our error and for an unrelated `template not found` too. |
| `twoverbs` | `%v` and `%w` side by side. Both print `profile 42: not found`, character for character. `errors.Unwrap` gives `<nil>` for the `%v` one and the original `not found` for the `%w` one. |
| `walk` | `==` against the `%w` error: false. `errors.Is` on the `%w` error: true. `errors.Is` on the `%v` error: false. |
| `fixed` | The hook with one letter changed, `%v` to `%w`: `404 profile 42: not found`. |
| `chain` | A custom error type, `QueryError`, with a `Table` field and an `Unwrap` method, three errors deep. A small loop walks the chain with `errors.Unwrap` and prints each link's type. |
| `as` | `errors.Is` still finds `ErrNotFound` three links down; `errors.As` finds the `*QueryError` and reads its table; a plain type assertion on the outer error is false. |
| `astype` | The same `errors.As` lookup with `errors.AsType`, in one line (Go 1.26+). |
| `nounwrap` | `QueryError` with its `Unwrap` method deleted. The message still prints in full (its `Error` method reads `Err` directly), `errors.As` still finds it, and `errors.Is` can no longer reach `ErrNotFound`. |
| `stdlib` | The standard library doing both: `strconv.Atoi("forty")` returns a `*strconv.NumError` wrapping `strconv.ErrSyntax`, so `errors.Is` and `errors.AsType` both work on it (Go 1.26+). |

Nothing here crashes or fails on purpose; every "wrong" answer is a program that runs fine and prints the wrong status, which is the point.
