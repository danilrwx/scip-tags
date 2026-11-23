# scip-tags

**Jump to the right definition in Vim, even when two symbols have the same name.**

[![CI](https://github.com/danilrwx/scip-tags/actions/workflows/ci.yml/badge.svg)](https://github.com/danilrwx/scip-tags/actions/workflows/ci.yml)
[![Go 1.26+](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go&logoColor=white)](go.mod)
[![Apache 2.0](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)

`scip-tags` turns a [SCIP](https://github.com/scip-code/scip) index into a
ctags-compatible `tags` file. Point at `worker.Label()` and press `Ctrl-]` to jump
to `Worker.Label` even if `Manager.Label` exists in the same project. Press
`Ctrl-T` to go back.

The indexer resolves symbols when you build the index. Vim reads the resulting
tags locally, without a running language server.

## Quick start

Install the converter with Go 1.26 or later.

```sh
go install github.com/danilrwx/scip-tags/cmd/scip-tags@latest
```

From your project root, generate `index.scip` with your language's indexer, then
convert it. For a Go project, for example:

```sh
go install github.com/scip-code/scip-go/cmd/scip-go@latest
scip-go
scip-tags > tags.tmp && mv tags.tmp tags
```

Open Vim in the project. Use `:tag Name` to jump by name and `:tselect Name` to
list matching definitions. The converter works with any language that has a
SCIP producer, including
[C/C++](https://github.com/sourcegraph/scip-clang),
[JavaScript and TypeScript](https://github.com/sourcegraph/scip-typescript), and
[Java](https://github.com/scip-code/scip-java).

### Jump from the cursor in Vim

Clone this repository and add the plugin to your vimrc:

```vim
source /path/to/scip-tags/vim/scip-tags.vim
```

Put the cursor on a symbol and press Vim’s built-in `Ctrl-]`. The plugin uses
the file, line and column to look up the indexed definition. Use `g Ctrl-]` to
choose between multiple targets and `Ctrl-T` to return. Vim looks up ordinary
tags by name when there is no position tag at the cursor. Regenerate the tags
after editing indexed source lines.

The plugin uses Vim's `tagfunc`. If you already configured a tag function, Vim
continues to use it.

## Reuse an index

`scip-tags` reads `index.scip` by default. Repeat `-scip` to combine indexes
for the same source tree, and use `-root` if the tree is at a different path
than the one recorded in the index.

```sh
scip-tags -root . -scip go.scip -scip web.scip > tags.tmp && mv tags.tmp tags
```

## Use from Go

The [`tags`](tags) package exposes `Write(w, source, indexes...)`. Pass an
`io.Writer`, an `fs.FS` containing the matching source tree, and one or more
decoded `*scip.Index` values.

## Tests and accuracy

Run the regression suite with `go test ./...`. It checks that same-name methods
and file-local symbols keep their own definitions, forward declarations do not
replace definitions, and merging indexes preserves both versioned symbols and
document-scoped local IDs. Other cases cover UTF-8, UTF-16 and UTF-32 coordinates,
multiline ranges, and generated files outside the source root.

CI runs the regression suite, `go vet`, and golangci-lint, then builds the CLI for
Linux and OpenBSD/ARM. A separate headless Vim check exercised `Ctrl-]` on two
same-name Go methods, `Ctrl-T` back to the call site, and rejection of a changed
source line.

### Definition benchmark

We checked the tags emitted by `scip-tags` at known reference positions in
`golang.org/x/mod`, Lua, and libevent. Each count below is a correct file and
line. The ctags baseline uses Vim's first name match. The SCIP columns count
unambiguous destinations from the generated tags.

| Project | References | Universal Ctags | Language server | Stock SCIP → tags | Repaired SCIP → tags |
| --- | ---: | ---: | ---: | ---: | ---: |
| `golang.org/x/mod`, within the module | 2,205 | 1,265 | 2,205 (gopls) | 2,202 | — |
| Lua 5.4.9 | 2,656 | 1,267 | 2,644 (clangd) | 1,750 | 2,017 |
| libevent 2.1.13 | 2,900 | 1,512 | 2,894 (clangd) | 1,777 | 2,295 |

The C cases include same-name fields and macro arguments. The repaired column
uses C indexes with macro-argument positions corrected before conversion.
With the project-only Go index, zero of 200 additional references to definitions
outside the module resolved to the expected location. Coverage depends on the
definitions and occurrence positions supplied by the input indexes.
The benchmark was run separately from CI.

## License

[Apache 2.0](LICENSE)
