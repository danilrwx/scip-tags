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

package index

import (
	"bytes"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/scip-code/scip/bindings/go/scip"
)

func TestTagsKeepHomonymsAndResolvePositions(t *testing.T) {
	const (
		definition = int32(scip.SymbolRole_Definition)
		forward    = int32(scip.SymbolRole_ForwardDefinition)
		worker     = "scip go test v1 Worker#Label()."
		manager    = "scip go test v1 Manager#Label()."
	)

	index := &scip.Index{Documents: []*scip.Document{
		{RelativePath: "a.go", Occurrences: []*scip.Occurrence{
			{Range: []int32{0, 0, 5}, Symbol: worker, SymbolRoles: definition},
			{Range: []int32{1, 0, 5}, Symbol: manager, SymbolRoles: definition | forward},
		}},
		{RelativePath: "b.go", Occurrences: []*scip.Occurrence{
			{Range: []int32{0, 0, 5}, Symbol: manager, SymbolRoles: definition},
		}},
		{
			RelativePath: "call.go", PositionEncoding: scip.PositionEncoding_UTF16CodeUnitOffsetFromLineStart,
			Occurrences: []*scip.Occurrence{{Range: []int32{0, 3, 8}, Symbol: worker}},
		},
	}}
	tree := fstest.MapFS{
		"a.go":    {Data: []byte("Label()\nLabel();\n")},
		"b.go":    {Data: []byte("Label()\n")},
		"call.go": {Data: []byte("🚀 Label()\n")},
	}

	var out bytes.Buffer
	if err := writeTags(&out, tree, index, index); err != nil {
		t.Fatal(err)
	}

	tags := out.String()
	if strings.Count(tags, "\nLabel\t") != 2 || strings.Contains(tags, "Label\ta.go\t2") {
		t.Fatalf("duplicate definitions or forward declaration: %s", tags)
	}

	if !strings.Contains(tags, "__scip__:call.go:1:6\ta.go\t1;\"\tcolumn:1\tend:11\tsourcehash:") ||
		strings.Contains(tags, "__scip__:call.go:1:6\tb.go") {
		t.Fatalf("wrong target or UTF-16 position: %s", tags)
	}
}
