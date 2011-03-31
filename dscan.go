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