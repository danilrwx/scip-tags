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

package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"

	"github.com/danilrwx/scip-tags/tags"
	"github.com/scip-code/scip/bindings/go/scip"
	"google.golang.org/protobuf/proto"
)

func readSCIP(path string) (*scip.Index, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	index := new(scip.Index)
	if err := proto.Unmarshal(data, index); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	return index, nil
}

func scipRoot(index *scip.Index) (string, error) {
	root, err := url.Parse(index.GetMetadata().GetProjectRoot())
	if err != nil || root.Scheme != "file" || (root.Host != "" && root.Host != "localhost") ||
		root.User != nil || root.RawQuery != "" || root.Fragment != "" || !filepath.IsAbs(root.Path) {
		return "", fmt.Errorf("SCIP project root must be an absolute local file URL")
	}

	return filepath.Clean(root.Path), nil
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
	defer func() { _ = tree.Close() }()

	return tags.Write(w, tree.FS(), indexes...)
}
