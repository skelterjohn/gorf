// Copyright 2011 John Asmuth. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"go/ast"
	"fmt"
	"rog-go.googlecode.com/hg/exp/go/types"
)

func ScanCmd(args []string) (err os.Error) {
	ScanAllForImports(LocalRoot)
	for _, path := range args {
		pkg := LocalImporter(path)
		
		for _, file := range pkg.Files {
			GetCommentTies(file)
		}
		
		ast.Walk(DepthWalker(0), pkg)
	}
	return
}

type DepthWalker int

func (this DepthWalker) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return this+1
	}
	
	buffer := ""
	for i:=0;i<int(this); i++ {
		buffer += " "
	}
	
	fmt.Printf("%sPos: %d %s\n", buffer, node.Pos(), AllSources.Position(node.Pos()))
	fmt.Printf("%s%T\n", buffer, node)
	fmt.Printf("%s%v\n", buffer, node)
	if e, ok := node.(ast.Expr); ok {
		obj, typ := types.ExprType(e, LocalImporter)
		fmt.Printf("%s%v\n", buffer, obj)
		fmt.Printf("%s%v\n", buffer, typ)
	}
	fmt.Println()
	
	switch n := node.(type) {
	
	}
	
	return this+1
}
