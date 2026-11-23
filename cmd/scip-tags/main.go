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
	"flag"
	"fmt"
	"log"
	"os"
)

func run() error {
	root := flag.String("root", "", "source root (default: SCIP project root)")
	format := flag.String("format", "tags", "output format (tags or ctags)")

	var paths []string

	flag.Func("scip", "SCIP index file (repeatable, default index.scip)", func(path string) error {
		paths = append(paths, path)
		return nil
	})

	flag.Parse()

	if *format != "tags" && *format != "ctags" {
		return fmt.Errorf("unknown output format %q (use tags)", *format)
	}

	if len(paths) == 0 {
		paths = []string{"index.scip"}
	}

	return exportTags(os.Stdout, *root, paths)
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
