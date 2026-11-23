# scip-tags

**Jump to the right definition in Vim, even when two symbols have the same name.**

[![CI](https://github.com/danilrwx/scip-tags/actions/workflows/ci.yml/badge.svg)](https://github.com/danilrwx/scip-tags/actions/workflows/ci.yml)
[![Go 1.26+](https://img.shields.io/badge/Go-1.26%2B-00ADD8?logo=go&logoColor=white)](go.mod)
[![Apache 2.0](https://img.shields.io/badge/license-Apache%202.0-blue)](LICENSE)

`scip-tags` turns a [SCIP](https://github.com/scip-code/scip) index into a
ctags-compatible `tags` file. Point at `worker.Label()`, press `gd`, and land on
`Worker.Label` even if `Manager.Label` exists in the same project. Press
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
nmap gd <Plug>(ScipTagsDefinition)
```

Put the cursor on a symbol and press `gd`. The plugin uses the file, line and
column to look up the indexed definition. If you edit the line containing the
symbol, regenerate the tags before jumping from that line again.

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

## License

[Apache 2.0](LICENSE)
