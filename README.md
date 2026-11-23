# scip-tags

`scip-tags` converts [SCIP](https://github.com/scip-code/scip) indexes into
ctags-compatible `tags` files. It reads the same SCIP format for C/C++, Go,
Python, TypeScript, and any other language with a SCIP producer. Generate the
index with your language's SCIP indexer before running the converter.

## Build tags

Build the CLI with Go 1.26 or later.

```sh
go build -o "$HOME/.local/bin/scip-tags" ./cmd/scip-tags
```

In the source root, write the tag file alongside the sources.

```sh
scip-tags -scip index.scip > tags.tmp && mv tags.tmp tags
```

The CLI reads `index.scip` by default. Repeat `-scip` for several indexes of
the same source tree. Source paths in the tag file are relative to that tree.
If the index was generated elsewhere, point `-root` at the matching sources.
Tags export also works outside a Git checkout.

Vim uses its built-in `Ctrl-]`, `:tag name`, and `:tselect name`. When
distinct definitions have the same name, the tag file preserves them as
separate entries. `:tselect` lists the candidates.

For an exact SCIP jump by cursor position, source `vim/scip-tags.vim` from
your vimrc and bind its command to `gd`. It reads the same `tags` file and
uses Vim's tag stack, so `Ctrl-T` returns to the call site.

```vim
source /path/to/scip-tags/vim/scip-tags.vim
nmap gd <Plug>(ScipTagsDefinition)
```

## Export a map for Patchtree

Patchtree uses a revision-pinned JSON map with occurrence ranges and resolved
targets. Export it from the clean Git checkout used to build the SCIP index.

```sh
scip-tags -format patchtree -repo https://github.com/owner/repo -scip index.scip > code-index.json
```

Load the map under **Settings → Code index → Import…** in Patchtree. JSON
mode joins references to exact SCIP symbols, including when names collide.
SCIP UTF-8, UTF-16 and UTF-32 positions are translated to UTF-8 byte offsets.
The previous usage with `-repo` also selects the Patchtree format.

## Embedded Go backend for Patchtree

Build the browser artifact from the same source and toolchain as the native
binary. Package both browser files with the extension.

```sh
GOOS=js GOARCH=wasm go build -o code-index.wasm ./cmd/scip-tags
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" .
```

To build Patchtree with this checkout, set
`PATCHTREE_CODE_INDEX_DIR=/path/to/scip-tags/cmd/scip-tags`.

After starting the module with the Go runtime in a Web Worker, call
`codeIndexGo(JSON.stringify(snapshot))`. It returns the navigation map as a JSON
string or a JSON object with an `error` string. Native tools can send the same
snapshot to `scip-tags -snapshot` on stdin and read the map from stdout.
Errors from the CLI go to stderr with a nonzero exit status.

A version 1 snapshot carries the exact repository revision and the target build
configuration. Supply source paths relative to the repository root, including
`go.mod`.

```json
{
  "version": 1,
  "repository": "https://github.com/owner/repo",
  "revision": "0123456789abcdef0123456789abcdef01234567",
  "goos": "linux",
  "goarch": "amd64",
  "tags": [],
  "sources": {
    "go.mod": "module example.com/repo\n",
    "example.go": "package example\nconst Label = \"example\"\nvar name = Label\n"
  }
}
```

The caller supplies source files from its checkout or repository API. The Go
backend checks the root module or the local modules listed in the root
`go.work`, including Go tests, with cgo disabled.
Unresolved imports, including standard library and external modules, appear in
`diagnostics`. References the checker resolves still appear in `files`.
Patchtree packages the WASM backend. It passes a GitHub or GitLab tar.gz
snapshot to `codeIndexGoArchive` and caches the resulting map by repository and
commit. The browser uses Linux/amd64 with default build tags and cgo disabled.
Dependency downloads and a versioned dependency cache are pending.

Patchtree imports maps up to 64 MiB, builds maps up to 128 MiB and downloads
archives up to 32 MiB compressed or 128 MiB expanded. Links use the review repository, so a
fork's head must be accessible there for links to open.
