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
	"bufio"
	"bytes"
	"cmp"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"slices"
	"strconv"
	"strings"
)

const (
	positionTagPrefix  = "__scip__:"
	referenceTagPrefix = "__scipref__:"
)

// Position identifies a location in the source tree using 1-based byte columns.
type Position struct {
	Path   string `json:"path"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

// Reference includes the source line for a matching position.
type Reference struct {
	Position
	Text string `json:"text"`
}

type positionTag struct {
	source    Position
	target    Position
	endColumn int
	hash      string
	symbol    string
}

// References resolves a position and finds the indexed uses of its targets.
// The hash is optional and represents the current source line in the editor.
func References(input io.ReadSeeker, tree fs.FS, query Position, hash string) ([]Reference, error) {
	if !validDocumentPath(query.Path) || query.Line < 1 || query.Column < 1 {
		return nil, errors.New("provide a source path, line and column")
	}

	sources := make(map[string][]string)
	isCurrent := func(pos Position, hash string) bool {
		lines, ok := sources[pos.Path]
		if !ok {
			data, err := fs.ReadFile(tree, pos.Path)
			if err == nil {
				lines = strings.Split(strings.ReplaceAll(string(data), "\r\n", "\n"), "\n")
			}

			sources[pos.Path] = lines
		}

		return pos.Line <= len(lines) && sourceHash(lines[pos.Line-1]) == hash
	}

	targets := make(map[string]struct{})
	cursorPrefix := []byte(query.Path + ":" + strconv.Itoa(query.Line) + ":")

	err := scanTags(input, func(line []byte, prefix string) {
		if !bytes.HasPrefix(line[len(prefix):], cursorPrefix) {
			return
		}

		entry, ok := parsePositionTag(line, prefix, nil)
		if !ok || query.Column < entry.source.Column || query.Column >= entry.endColumn ||
			hash != "" && hash != entry.hash || !isCurrent(entry.source, entry.hash) {
			return
		}

		key := entry.symbol
		if key == "" {
			key = fmt.Sprintf("%s\t%d;\"\tcolumn:%d", entry.target.Path, entry.target.Line, entry.target.Column)
		}

		targets[key] = struct{}{}
	})
	if err != nil {
		return nil, err
	}

	if len(targets) == 0 {
		return nil, errors.New("no current SCIP position at the cursor")
	}

	if _, err := input.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	found := make(map[Position]struct{})

	err = scanTags(input, func(line []byte, prefix string) {
		entry, ok := parsePositionTag(line, prefix, targets)
		if !ok || entry.source == entry.target {
			return
		}

		if isCurrent(entry.source, entry.hash) {
			found[entry.source] = struct{}{}
		}
	})
	if err != nil {
		return nil, err
	}

	results := make([]Reference, 0, len(found))
	for pos := range found {
		results = append(results, Reference{Position: pos, Text: strings.TrimSpace(sources[pos.Path][pos.Line-1])})
	}

	slices.SortFunc(results, func(a, b Reference) int {
		return cmp.Or(cmp.Compare(a.Path, b.Path), cmp.Compare(a.Line, b.Line), cmp.Compare(a.Column, b.Column))
	})

	return results, nil
}

func scanTags(input io.Reader, visit func([]byte, string)) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 64<<10), 1<<20)

	for scanner.Scan() {
		line := scanner.Bytes()
		switch {
		case bytes.HasPrefix(line, []byte(positionTagPrefix)):
			visit(line, positionTagPrefix)
		case bytes.HasPrefix(line, []byte(referenceTagPrefix)):
			visit(line, referenceTagPrefix)
		}
	}

	return scanner.Err()
}

func parsePositionTag(line []byte, prefix string, targets map[string]struct{}) (positionTag, bool) {
	var fields [6][]byte

	raw := line
	for i := range len(fields) - 1 {
		field, rest, ok := bytes.Cut(raw, []byte("\t"))
		if !ok {
			return positionTag{}, false
		}

		fields[i], raw = field, rest
		if i == 3 && targets != nil {
			key := line[len(fields[0])+1 : len(line)-len(rest)-1]
			if prefix == referenceTagPrefix {
				key = field
			}

			if _, matched := targets[string(key)]; !matched {
				return positionTag{}, false
			}
		}
	}

	fields[5] = raw
	source := fields[0][len(prefix):]

	colStart := bytes.LastIndexByte(source, ':')
	if colStart < 0 {
		return positionTag{}, false
	}

	rowStart := bytes.LastIndexByte(source[:colStart], ':')
	if rowStart < 0 {
		return positionTag{}, false
	}

	sourceLine := positiveNumber(source[rowStart+1 : colStart])
	sourceColumn := positiveNumber(source[colStart+1:])
	endColumn := positiveNumber(bytes.TrimPrefix(fields[4], []byte("end:")))
	hash, hashOK := bytes.CutPrefix(fields[5], []byte("sourcehash:"))
	sourcePath := string(source[:rowStart])

	if sourceLine == 0 || sourceColumn == 0 || endColumn <= sourceColumn || !hashOK || !validDocumentPath(sourcePath) {
		return positionTag{}, false
	}

	entry := positionTag{
		source:    Position{sourcePath, sourceLine, sourceColumn},
		endColumn: endColumn,
		hash:      string(hash),
	}

	if prefix == referenceTagPrefix {
		if id, ok := bytes.CutPrefix(fields[3], []byte("symbol:")); !ok || positiveNumber(id) == 0 {
			return positionTag{}, false
		}

		entry.symbol = string(fields[3])

		return entry, true
	}

	targetPath := string(fields[1])
	targetLine := positiveNumber(bytes.TrimSuffix(fields[2], []byte(";\"")))

	targetColumn := positiveNumber(bytes.TrimPrefix(fields[3], []byte("column:")))
	if targetLine == 0 || targetColumn == 0 || !validDocumentPath(targetPath) {
		return positionTag{}, false
	}

	entry.target = Position{targetPath, targetLine, targetColumn}

	return entry, true
}

func positiveNumber(value []byte) int {
	number, err := strconv.Atoi(string(value))
	if err != nil || number < 1 {
		return 0
	}

	return number
}
