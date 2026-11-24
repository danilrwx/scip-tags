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
	"fmt"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/scip-code/scip/bindings/go/scip"
)

func TestReferencesReadLegacyPositionTags(t *testing.T) {
	tree := fstest.MapFS{
		"declaration.go": {Data: []byte("func Test() {}\n")},
		"use.go":         {Data: []byte("    Test()\n")},
	}

	text := fmt.Sprintf(`__scip__:declaration.go:1:6	declaration.go	1;"	column:6	end:10	sourcehash:%s
__scip__:use.go:1:5	declaration.go	1;"	column:6	end:9	sourcehash:%s
__scip__:../outside.go:1:5	declaration.go	1;"	column:6	end:9	sourcehash:%s
__scip__:use.go:1:999999999999999999999	declaration.go	1;"	column:6	end:9	sourcehash:%s
`, sourceHash("func Test() {}"), sourceHash("    Test()"), sourceHash("    Test()"), sourceHash("    Test()"))

	got, err := References(strings.NewReader(text), tree, Position{"declaration.go", 1, 6}, "")

	want := []Reference{{Position: Position{"use.go", 1, 5}, Text: "Test()"}}
	if err != nil || !slices.Equal(got, want) {
		t.Fatalf("legacy references = %+v, %v; want %+v", got, err, want)
	}
}

func TestReferencesKeepSymbolsWithTheSameDefinitionPositionSeparate(t *testing.T) {
	const (
		typeSymbol        = "scip-ts npm app v1 Definition#"
		constructorSymbol = "scip-ts npm app v1 Definition."
		definition        = int32(scip.SymbolRole_Definition)
	)

	tree := fstest.MapFS{
		"model.ts":  {Data: []byte("class Definition {}\n")},
		"types.ts":  {Data: []byte("type Options = Definition;\n")},
		"values.ts": {Data: []byte("new Definition();\n")},
	}
	index := &scip.Index{Documents: []*scip.Document{
		{RelativePath: "model.ts", Occurrences: []*scip.Occurrence{
			{TypedRange: scip.NewRangeUnchecked([]int32{0, 6, 16}).AsTypedRange(), Symbol: typeSymbol, SymbolRoles: definition},
			{TypedRange: scip.NewRangeUnchecked([]int32{0, 6, 16}).AsTypedRange(), Symbol: constructorSymbol, SymbolRoles: definition},
		}},
		{RelativePath: "types.ts", Occurrences: []*scip.Occurrence{
			{TypedRange: scip.NewRangeUnchecked([]int32{0, 15, 25}).AsTypedRange(), Symbol: typeSymbol},
		}},
		{RelativePath: "values.ts", Occurrences: []*scip.Occurrence{
			{TypedRange: scip.NewRangeUnchecked([]int32{0, 4, 14}).AsTypedRange(), Symbol: constructorSymbol},
		}},
	}}

	var out bytes.Buffer
	if err := Write(&out, tree, index); err != nil {
		t.Fatal(err)
	}

	typed := Reference{Position: Position{Path: "types.ts", Line: 1, Column: 16}, Text: "type Options = Definition;"}
	constructed := Reference{Position: Position{Path: "values.ts", Line: 1, Column: 5}, Text: "new Definition();"}

	for _, test := range []struct {
		query Position
		want  []Reference
	}{
		{typed.Position, []Reference{typed}},
		{constructed.Position, []Reference{constructed}},
		{Position{Path: "model.ts", Line: 1, Column: 7}, []Reference{typed, constructed}},
	} {
		got, err := References(bytes.NewReader(out.Bytes()), tree, test.query, "")
		if err != nil || !slices.Equal(got, test.want) {
			t.Errorf("references at %+v = %+v, %v; want %+v", test.query, got, err, test.want)
		}
	}
}

func TestMultilineOccurrenceReturnsOneReference(t *testing.T) {
	const symbol = "scip-clang cpp sample v1 Handler()."

	tree := fstest.MapFS{
		"handler.h": {Data: []byte("void Handler();\n")},
		"macro.c":   {Data: []byte("HANDLER(\n  ctx,\n  flags);\n")},
	}
	index := &scip.Index{Documents: []*scip.Document{
		{RelativePath: "handler.h", Occurrences: []*scip.Occurrence{
			{TypedRange: scip.NewRangeUnchecked([]int32{0, 5, 12}).AsTypedRange(), Symbol: symbol, SymbolRoles: int32(scip.SymbolRole_Definition)},
		}},
		{RelativePath: "macro.c", Occurrences: []*scip.Occurrence{
			{TypedRange: scip.NewRangeUnchecked([]int32{0, 0, 2, 7}).AsTypedRange(), Symbol: symbol},
		}},
	}}

	var out bytes.Buffer
	if err := Write(&out, tree, index); err != nil {
		t.Fatal(err)
	}

	want := []Reference{{Position: Position{Path: "macro.c", Line: 1, Column: 1}, Text: "HANDLER("}}

	for _, query := range []Position{{"handler.h", 1, 6}, {"macro.c", 1, 3}, {"macro.c", 2, 4}, {"macro.c", 3, 6}} {
		got, err := References(bytes.NewReader(out.Bytes()), tree, query, "")
		if err != nil || !slices.Equal(got, want) {
			t.Errorf("references at %+v = %+v, %v; want %+v", query, got, err, want)
		}
	}
}
