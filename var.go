package main

import (
	"fmt"
	"go/ast"
)

type ChangeLocalVarWalker struct {
	oldident, newident string
	changed *bool
	redefined bool
	nested bool
}

func (w *ChangeLocalVarWalker) Visit(node ast.Node) (v ast.Visitor) {

	if false && node != nil && !w.nested {
		fmt.Printf("%T\n%v\n\n", node, node)
	}

	if w.redefined {
		return nil
	}
	
	if w.nested && NodeRedefinesIdent(node, w.oldident, w) {
		w.redefined = true
		return nil
	}
	
	switch n := node.(type) {
	case *ast.BlockStmt:
		changer := &ChangeLocalVarWalker{}
		*changer = *w
		changer.nested = true
		return changer
	case *ast.FuncDecl:
		changer := &ChangeLocalVarWalker{}
		*changer = *w
		changer.nested = true
		return changer
	case *ast.ValueSpec:
		for _, name := range n.Names {
			if name.Name == w.oldident {
				name.Name = w.newident
				*w.changed = true
			}
		}
	case *ast.SelectorExpr:
		// it's not a global ident. since we're local we don't care
		return nil
	case *ast.Ident:
		if n.Name == w.oldident {
			n.Name = w.newident
			*w.changed = true
		}
	}
	
	return w
}