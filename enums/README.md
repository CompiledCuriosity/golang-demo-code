# Go enums: iota, a type, and String

Source for [Go Enums: Why Your New User Is an Admin (iota Explained)](https://youtu.be/hU4rLlnw3Fc).

Every program here is the one that produced the output shown in the video. The job is always the same small example: a `User` with a `Role`, and the two ways an enum built from an unchecked int quietly gets the wrong answer.

Related: [`pipeline-pattern`](../pipeline-pattern), [`worker-pool`](../worker-pool) and [`concurrency-patterns`](../concurrency-patterns) are unrelated to enums but share this repo's run command.

Run any of them from the repo root:

```
go run ./enums/hook
```

## The programs, in the order the video walks them

| Folder | What it shows |
| --- | --- |
| `hook` | The cold open. A `User` gets a name and no `Role`. Printing it says `sam Admin` - nobody made Sam an admin. |
| `magic` | A role as a bare `int`, checked with a magic number: `if role == 0`. Compiles and runs; means nothing to read. |
| `byhand` | The same check with named constants, numbered by hand: `Admin, Editor, Viewer = 0, 1, 2`. |
| `iotalong` | The same three constants, numbered by `iota` written on every line, to show what `iota` is standing in for. |
| `iotashort` | The short form: bare `Editor` and `Viewer` repeat `= iota`, and a second `const` block restarts the counter at zero. |
| `swap` | Plain int constants still mix with any other int: a call with its arguments swapped compiles and runs, and gives the wrong user the wrong role. |
| `typed` | `type Role int` on the same constants. The same swapped call is now refused twice by the compiler - see "Meant to fail" below. |
| `typedok` | The call fixed, arguments the right way round. Builds and runs; prints the role as a bare number, because there is no `String` method yet. |
| `named` | A `String` method on `Role`. Same call as `typedok`; `fmt` finds the method itself and the print says `Editor`, not `1`. |
| `unknown` | Hole one, fixed. `Unknown` is named first in the block, so an unset `Role` (Sam's, again) prints `Unknown` instead of landing on a real name. |
| `noconvert` | An `int` from outside assigned straight to a `Role` variable. Meant to fail - see below. |
| `convert` | The same value, explicitly converted: `Role(fromDB)`. Compiles and runs even though there is no such role, and prints the fallback `Role(42)`. |
| `literal` | The second way in: a bare literal typed straight into a call, `grant(7, 42)`. No conversion needed, no complaint from the compiler. |
| `isvalid` | The fix for both: an `IsValid` range check, tried against 42, an unset `Role`, and a real one. |
| `stringenum` | The alternative asked about at the end: `type Role string`. An unset value is the empty string, but any string still converts, same as any number did. |

## Meant to fail

`typed` and `noconvert` do not build. That is the point of them.

`typed` is `go build ./enums/typed` failing with two errors on the same swapped call:

```
./main.go:19:8: cannot use Editor (constant 1 of int type Role) as int value in argument to grant
./main.go:19:16: cannot use id (variable of type int) as Role value in argument to grant
```

`noconvert` is one error, an `int` variable assigned to a `Role` variable with no conversion:

```
./main.go:9:18: cannot use fromDB (variable of type int) as Role value in variable declaration
```

The exact wording above is what go1.27.1 prints (`constant 1 of int type Role`); older toolchains phrase the first line differently (`constant 1 of type Role`, with no `int`). Either way, both programs are refused, which is the only thing the video claims.

## About the numbers

Nothing here is timed or concurrent - every program is deterministic and was run three times with byte-identical output while making the video. The only thing that can legitimately differ by Go version is the exact wording of the two compiler errors above.
