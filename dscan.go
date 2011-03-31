// Copyright 2011 John Asmuth. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"go/ast"
)

func ScanCmd(args []string) (err os.Error) {
	ScanAllForImports(LocalRoot)
	for _, path := range args {
		pkg := LocalImporter(path)
		ast.Walk(DepthWalker(0), pkg)
	}
	return
}