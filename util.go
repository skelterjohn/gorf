// Copyright 2011 John Asmuth. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"strings"
	"unicode"
	"utf8"
	"go/ast"
	"fmt"
	"rog-go.googlecode.com/hg/exp/go/types"
)

func IsLegalIdentifier(s string) bool {
	us := utf8.NewString(s)
	if !unicode.IsLetter(us.At(0)) {
		return false
	}
	for i, c := range s {
		if !unicode.IsLetter(c) && (i == 0 || !unicode.IsDigit(c)) {
			return false
		}
	}
	return true
}

func QuotePath(path string) (qpath string) {
	qpath = "\""+path+"\""
	return
}

func TrimPath(path string) (tpath string) {
	tpath = strings.Trim(path, "\"")
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

func GetUniqueIdent(files []*ast.File, candidate string) (id string) {
	ic := make(IdentCollector)
	for _, file := range files {
		ast.Walk(ic, file)
	}
	
	id = candidate
	for i:=0; ic[id]; i++ {
		id = fmt.Sprintf("%s_%d", candidate, i)
	}
	
	return
}

type IdentCollector map[string]bool

func (this IdentCollector) Visit(node ast.Node) ast.Visitor {
	if ident, ok := node.(*ast.Ident); ok {
		this[ident.Name] = true
	}
	return this
}
