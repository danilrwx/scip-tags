// Copyright 2026 Daniil Antoshin
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build !js

package main

import (
	"bufio"
	"cmp"
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/scip-code/scip/bindings/go/scip"
)

type tag struct {
	name string
	location
	local bool
	end   int
	hash  string
}

func exportTags(w io.Writer, root string, paths []string) error {
	var indexes []*scip.Index
	for _, path := range paths {
		index, err := readSCIP(path)
		if err != nil {
			return err
		}
		indexes = append(indexes, index)
	}
	if root == "" {
		for i, index := range indexes {
			project, err := scipRoot(index)
			if err != nil {
				return fmt.Errorf("%s: %w (or specify -root)", paths[i], err)
			}
			if root != "" && root != project {
				return fmt.Errorf("%s: SCIP project root differs from %s", paths[i], root)
			}
			root = project
		}
	}
	tree, err := os.OpenRoot(root)
	if err != nil {
		return err
	}
	defer tree.Close()
	return writeTags(w, tree.FS(), indexes...)
}

func writeTags(w io.Writer, tree fs.FS, indexes ...*scip.Index) error {
	entries := make(map[tag]struct{})
	sources := make(map[string][]string)
	for _, index := range indexes {
		for _, doc := range index.GetDocuments() {
			path := doc.GetRelativePath()
			if !fs.ValidPath(path) || path == "." || strings.ContainsRune(path, '\\') || strings.ContainsFunc(path, unicode.IsControl) {
				return fmt.Errorf("unsupported SCIP document path %q", path)
			}
			lines := sources[path]
			if lines == nil {
				data, err := fs.ReadFile(tree, path)
				if err != nil {
					return fmt.Errorf("read SCIP document %q: %w", path, err)
				}
				if !utf8.Valid(data) {
					return fmt.Errorf("%s: source is not UTF-8", path)
				}
				lines = strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
				sources[path] = lines
			}
			for _, occ := range doc.GetOccurrences() {
				if !isDefinition(occ) {
					continue
				}
				source, ok := occ.SourceRange()
				if !ok {
					continue
				}
				start, startOK := byteOffset(lines, source.Start, doc.GetPositionEncoding())
				end, endOK := byteOffset(lines, source.End, doc.GetPositionEncoding())
				if !startOK || !endOK || source.End.Line < source.Start.Line {
					continue
				}
				name := ""
				if source.End.Line == source.Start.Line {
					if end <= start {
						continue
					}
					name = strings.TrimSpace(lines[source.Start.Line][start:end])
				} else if symbol, err := scip.ParseSymbol(occ.GetSymbol()); err == nil && len(symbol.Descriptors) > 0 {
					name = symbol.Descriptors[len(symbol.Descriptors)-1].Name
				}
				if name == "" || strings.ContainsRune(name, '\\') || strings.ContainsFunc(name, unicode.IsControl) {
					continue
				}
				entries[tag{name: name, location: location{path, int(source.Start.Line) + 1, start},
					local: strings.HasPrefix(occ.GetSymbol(), "local ")}] = struct{}{}
			}
		}
	}
	if len(entries) == 0 {
		return fmt.Errorf("no definitions in SCIP documents")
	}
	var lastPath, lineHash string
	var lastLine int
	visitSCIPSpans(sources, indexes, func(path string, line int, text string, span span) {
		name := fmt.Sprintf("__scip__:%s:%d:%d", path, line, span.Start+1)
		if path != lastPath || line != lastLine {
			hash := sha256.Sum256([]byte(text))
			lastPath, lastLine, lineHash = path, line, fmt.Sprintf("%x", hash[:6])
		}
		for _, target := range span.Targets {
			entries[tag{name: name, location: target, end: span.End + 1, hash: lineHash}] = struct{}{}
		}
	})
	tags := make([]tag, 0, len(entries))
	for entry := range entries {
		tags = append(tags, entry)
	}
	slices.SortFunc(tags, func(a, b tag) int {
		return cmp.Or(cmp.Compare(a.name, b.name), cmp.Compare(a.Path, b.Path),
			cmp.Compare(a.Line, b.Line), cmp.Compare(a.Column, b.Column), cmp.Compare(a.end, b.end),
			cmp.Compare(localField(a), localField(b)))
	})

	out := bufio.NewWriter(w)
	fmt.Fprintln(out, "!_TAG_FILE_FORMAT\t2\t/extended format/")
	fmt.Fprintln(out, "!_TAG_FILE_SORTED\t1\t/bytewise sorted/")
	fmt.Fprintln(out, "!_TAG_PROGRAM_NAME\tcode-index\t//")
	for _, entry := range tags {
		fmt.Fprintf(out, "%s\t%s\t%d;\"\tcolumn:%d%s", entry.name, entry.Path, entry.Line, entry.Column+1, localField(entry))
		if entry.end > 0 {
			fmt.Fprintf(out, "\tend:%d\tsourcehash:%s", entry.end, entry.hash)
		}
		fmt.Fprintln(out)
	}
	return out.Flush()
}

func localField(entry tag) string {
	if entry.local {
		return "\tfile:"
	}
	return ""
}
