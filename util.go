package main

import (
	"fmt"
	"go/ast"
	"go/token"
)

func QuoteTarget(target string) (qt string) {
	return fmt.Sprintf("\"%s\"", target)
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
