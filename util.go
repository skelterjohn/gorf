package main

import (
	"fmt"
	"path/filepath"
	"go/ast"
	"go/token"
	"unicode"
	"utf8"
)

func IsLegalIdentifier(id string) bool {
	if len(id) == 0 {
		return false
	}
	if !unicode.IsLetter(utf8.NewString(id).At(0)) {
		return false
	}
	for _, c := range id[1:] {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

func QuoteTarget(target string) (qt string) {
	return fmt.Sprintf("\"%s\"", filepath.Clean(target))
}

/*
An ident can be redefined by
	- a GenDecl
	- a := assignment
	- a function's parameter or result name
*/
func NodeRedefinesIdent(node ast.Node, ident string, rhsv ast.Visitor) (redefined bool) {
	switch n := node.(type) {
	case *ast.AssignStmt:
		if n.Tok == token.DEFINE {
			for _, e := range n.Lhs {
				if id, ok := e.(*ast.Ident); ok {
					if id.Name == ident {
						redefined = true
					}
				}
			}
		}
		
		if redefined {
			for _, rhsx := range n.Rhs {
				ast.Walk(rhsv, rhsx)
			}
		}
		
	case *ast.FuncType:
		if n.Params != nil {
			for _, param := range n.Params.List {
				for _, name := range (*param).Names {
					if name.Name == ident {
						redefined = true
					}
				}
			}
		}
		if n.Results != nil {
			for _, result := range n.Results.List {
				for _, name := range (*result).Names {
					if name.Name == ident {
						redefined = true
					}
				}		
			}
		}
	
	case *ast.GenDecl:
		for _, spec := range n.Specs {
			if vs, ok := spec.(*ast.ValueSpec); ok {
				for _, id := range vs.Names {
					if id.Name == ident {
						redefined = true
					}
				}
			}
		}
	}
	return
}
