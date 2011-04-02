// Copyright 2011 John Asmuth. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"strings"
	"unicode"
	"utf8"
	"go/ast"
	"path/filepath"
	"fmt"
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

func GetUniqueFilename(pkg *ast.Package, candidate string) (fname string) {
	ext := filepath.Ext(candidate)
	base := candidate[:len(candidate)-len(ext)]

	fname = candidate
	for i:=0; pkg.Files[fname] != nil; i++ {
		fname = fmt.Sprintf("%s_%d%s", base, i, ext)
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
