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

package tags

import (
	"bytes"
	"cmp"
	"slices"
	"strings"
	"testing"
	"testing/fstest"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/scip-code/scip/bindings/go/scip"
)

type matrixSymbol struct {
	scipID   string
	identity string
}

type matrixToken struct {
	matrixSymbol
	roles int32
}

type matrixSite struct {
	Reference
	identity   string
	definition bool
}

type matrixDocument struct {
	scip.Document
	lines []string
	sites []matrixSite
}

func matrixOffset(text string, enc scip.PositionEncoding) int32 {
	switch enc {
	case scip.PositionEncoding_UTF16CodeUnitOffsetFromLineStart:
		return int32(len(utf16.Encode([]rune(text))))
	case scip.PositionEncoding_UTF32CodeUnitOffsetFromLineStart:
		return int32(utf8.RuneCountInString(text))
	default:
		return int32(len(text))
	}
}

func (doc *matrixDocument) appendLine(prefix string, tokens ...matrixToken) {
	const name = "Read"

	line := prefix
	row := int32(len(doc.lines))
	first := len(doc.sites)

	for _, token := range tokens {
		startByte := len(line)
		start := matrixOffset(line, doc.PositionEncoding)
		line += name
		end := matrixOffset(line, doc.PositionEncoding)
		doc.Occurrences = append(doc.Occurrences, &scip.Occurrence{
			TypedRange: scip.NewRangeUnchecked([]int32{row, start, end}).AsTypedRange(),
			Symbol:     token.scipID, SymbolRoles: token.roles,
		})
		doc.sites = append(doc.sites, matrixSite{
			Reference:  Reference{Position: Position{doc.RelativePath, int(row) + 1, startByte + 1}},
			identity:   token.identity,
			definition: token.roles == int32(scip.SymbolRole_Definition),
		})
		line += " + "
	}

	doc.lines = append(doc.lines, line)
	for i := first; i < len(doc.sites); i++ {
		doc.sites[i].Text = strings.TrimSpace(line)
	}
}

func TestReferencesMatrix(t *testing.T) {
	dialects := []struct {
		name, ext, manager, prefix string
	}{
		{"c", ".c", "clang", "a->"},
		{"cpp", ".cpp", "clang", "bucket."},
		{"go", ".go", "gomod", "reader."},
		{"typescript", ".ts", "npm", "value?."},
		{"java", ".java", "maven", "service."},
	}
	encodings := []struct {
		name string
		id   scip.PositionEncoding
	}{
		{"utf8", scip.PositionEncoding_UTF8CodeUnitOffsetFromLineStart},
		{"utf16", scip.PositionEncoding_UTF16CodeUnitOffsetFromLineStart},
		{"utf32", scip.PositionEncoding_UTF32CodeUnitOffsetFromLineStart},
	}
	layouts := []struct {
		name, dir, prefix, newline string
	}{
		{"plain", "src", "", "\n"},
		{"astral", "dir with spaces", "/* 🚀 𝄞 */ ", "\n"},
		{"accent", "src:generated", "/* é é 漢 */ ", "\r\n"},
		{"tabs", "nested/one,two", "\t/* 世😀 */\t", "\r\n"},
	}
	worktreeStates := []string{"current", "edited-line", "deleted-file"}

	queries := 0

	for _, dialect := range dialects {
		for rotation, enc := range encodings {
			for variant, layout := range layouts {
				for _, state := range worktreeStates {
					t.Run(strings.Join([]string{dialect.name, enc.name, layout.name, state}, "/"), func(t *testing.T) {
						pkg := "scip-test " + dialect.manager + " sample v1 "
						api := matrixSymbol{pkg + "Primary#Read().", "primary"}
						homonym := matrixSymbol{pkg + "Secondary#Read().", "secondary"}
						v1 := matrixSymbol{"scip-test " + dialect.manager + " dependency v1 Read#", "dep-v1"}
						v2 := matrixSymbol{"scip-test " + dialect.manager + " dependency v2 Read#", "dep-v2"}
						localA := matrixSymbol{"local 0", "first-local"}
						localB := matrixSymbol{"local 0", "second-local"}

						var docs [4]matrixDocument
						for i, stem := range []string{"first", "second", "caller", "helper"} {
							docs[i].RelativePath = layout.dir + "/" + stem + dialect.ext
							docs[i].PositionEncoding = encodings[(rotation+i)%len(encodings)].id
						}

						prefix := layout.prefix + dialect.prefix
						def := int32(scip.SymbolRole_Definition)
						forward := def | int32(scip.SymbolRole_ForwardDefinition)
						docs[0].appendLine(prefix, matrixToken{api, def}, matrixToken{localA, def})
						docs[0].appendLine(prefix, matrixToken{localA, 0}, matrixToken{homonym, 0})
						docs[1].appendLine(prefix, matrixToken{homonym, def}, matrixToken{localB, def})
						docs[1].appendLine(prefix, matrixToken{localB, 0}, matrixToken{api, 0})
						docs[1].appendLine(prefix, matrixToken{api, forward})
						docs[2].appendLine(prefix, matrixToken{api, 0}, matrixToken{homonym, 0}, matrixToken{v1, 0}, matrixToken{v2, 0})
						docs[2].appendLine(prefix, matrixToken{v1, 0}, matrixToken{homonym, 0}, matrixToken{api, 0})
						docs[3].appendLine(prefix, matrixToken{v2, 0}, matrixToken{api, 0}, matrixToken{homonym, 0}, matrixToken{v1, 0})
						docs[3].appendLine(prefix, matrixToken{v2, 0}, matrixToken{api, 0})
						docs[3].appendLine(prefix, matrixToken{api, def})

						tree := fstest.MapFS{}

						var allSites []matrixSite

						for i := range docs {
							doc := &docs[i]
							tree[doc.RelativePath] = &fstest.MapFile{
								Data: []byte(strings.Join(doc.lines, layout.newline) + layout.newline),
							}
							allSites = append(allSites, doc.sites...)
						}

						first := &scip.Index{Documents: []*scip.Document{&docs[0].Document, &docs[3].Document}}
						second := &scip.Index{Documents: []*scip.Document{&docs[1].Document, &docs[2].Document}}

						indexes := []*scip.Index{first, second}
						if variant%2 != 0 {
							indexes = []*scip.Index{second, first, second}
						}

						var out bytes.Buffer
						if err := Write(&out, tree, indexes...); err != nil {
							t.Fatal(err)
						}

						isStale := func(pos Position) bool {
							return state == "deleted-file" && pos.Path == docs[3].RelativePath ||
								state == "edited-line" && pos.Path == docs[2].RelativePath && pos.Line == 2
						}

						switch state {
						case "deleted-file":
							delete(tree, docs[3].RelativePath)
						case "edited-line":
							tree[docs[2].RelativePath].Data = []byte(docs[2].lines[0] + layout.newline + "// changed")
						}

						// The oracle comes from the fixture's semantic labels and saved byte
						// offsets. It never reads the emitted tag records.
						expected := make(map[string][]Reference)

						for _, site := range allSites {
							if !site.definition && !isStale(site.Position) {
								expected[site.identity] = append(expected[site.identity], site.Reference)
							}
						}

						for _, refs := range expected {
							slices.SortFunc(refs, func(a, b Reference) int {
								return cmp.Or(cmp.Compare(a.Path, b.Path), cmp.Compare(a.Line, b.Line), cmp.Compare(a.Column, b.Column))
							})
						}

						for _, site := range allSites {
							for offset := range 3 {
								pos := site.Position
								pos.Column += []int{0, 2, 4}[offset]
								got, err := References(bytes.NewReader(out.Bytes()), tree, pos, "")
								queries++

								if offset == 2 || isStale(pos) {
									if err == nil {
										t.Errorf("position %+v should be unindexed; got %+v", pos, got)
									}

									continue
								}

								if err != nil || !slices.Equal(got, expected[site.identity]) {
									t.Errorf("position %+v, symbol %s: got %+v, %v; want %+v",
										pos, site.identity, got, err, expected[site.identity])
								}
							}
						}
					})
				}
			}
		}
	}

	t.Logf("%d generated reference queries", queries)
}
