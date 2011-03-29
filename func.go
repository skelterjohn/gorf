package main

import (
	"go/ast"
)

type ChangeLocalFuncWalker struct {
	oldident, newident string
	changed, exists *bool
	redefined bool
}

func (w *ChangeLocalFuncWalker) Visit(node ast.Node) (v ast.Visitor) {
	if w.redefined {
		return nil
	}
	if NodeRedefinesIdent(node, w.oldident, w) {
		w.redefined = true
		return nil
	}
	
	switch n := node.(type) {
	case *ast.BlockStmt:
		changer := &ChangeLocalFuncWalker{}
		*changer = *w
		return changer
	case *ast.FuncDecl:
		if n.Name.Name == w.newident {
			*w.exists = true
		}
		if n.Name.Name == w.oldident {
			n.Name.Name = w.newident
			*w.changed = true
		}
		changer := &ChangeLocalFuncWalker{}
		*changer = *w
		return changer
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