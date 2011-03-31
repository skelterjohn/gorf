// Copyright 2011 John Asmuth. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"go/ast"
	"rog-go.googlecode.com/hg/exp/go/types"
)

func RenameInAll(path string, newname string, Obj *ast.Object) (err os.Error) {
	for _, ip := range ImportedBy[QuotePath(path)] {
		ipkg := LocalImporter(ip)
		for fpath, file := range ipkg.Files {
			if RenameInFile(file, newname, Obj) {
				err = RewriteSource(fpath, file)
				if err != nil {
					return
				}
			}
		}
	}
	return
}

func RenameInFile(file *ast.File, NewName string, Obj *ast.Object) bool {
	rw := RenameWalker{NewName:NewName, Obj:Obj}
	ast.Walk(&rw, file)
	return rw.Updated
}

type RenameWalker struct {
	NewName string
	Updated bool
	Obj *ast.Object
}

func (this *RenameWalker) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.Ident:
		obj, _ := types.ExprType(n, LocalImporter)
		if obj == this.Obj {
			this.Updated = true
			n.Name = this.NewName
		}
	case *ast.SelectorExpr:
		obj, _ := types.ExprType(n, LocalImporter)
		if obj == this.Obj {
			this.Updated = true
			n.Sel.Name = this.NewName
		}
	}
	return this
}