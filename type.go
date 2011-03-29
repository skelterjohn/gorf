package main

import (
	"os"
	"fmt"
	"path/filepath"
	"go/ast"
)


func ChangeType(target, pkgname, oldname, newname string) (err os.Error) {
	ScanForTargets()
	
	targ, ok := GetDirTargets(target)[pkgname]
	if !ok {
		err = os.NewError(fmt.Sprintf("No package %s in '%s'", pkgname, target))
		return
	}
	
	changed := false
	for _, src := range targ.Source {
		fpath := filepath.Join(targ.Path, src)
		if ft, ok := AllSources[fpath]; ok {
			clt := ChangeTypeWalker		{
				oldident:oldname, newident:newname,
				changed:new(bool),
			}
			ast.Walk(&clt, ft)
			if *clt.changed {
				changed = true
				RewriteSource(fpath, ft)
			}
		} else {
			panic("never happens")
		}
	}

	if changed {
	
		for fpath, ft := range AllSources {
			_ = fpath	
			ciw := CheckImportWalker{target:target}
			ast.Walk(&ciw, ft)
			if ciw.used {
				if ciw.pkgname == "" {
					ciw.pkgname = pkgname
				}
				clt := ChangeTypeWalker {
					pkgname:pkgname,
					oldident:oldname, newident:newname,
					changed:new(bool),
				}
				ast.Walk(&clt, ft)
				if *clt.changed {
					changed = true
					RewriteSource(fpath, ft)
				}
			}
		}
	}
	
	return
}

type ChangeTypeWalker struct {
	pkgname string
	oldident, newident string
	changed *bool
}

func (w *ChangeTypeWalker) Visit(node ast.Node) (v ast.Visitor) {

	if false && node != nil {
		fmt.Printf("%T\n%v\n\n", node, node)
	}
	
	switch n := node.(type) {
	case *ast.TypeSpec:
		if w.pkgname == "" {
			if n.Name.Name == w.oldident {
				n.Name.Name = w.newident
				*w.changed = true
			}
		}
	case *ast.ValueSpec:
		if w.pkgname == "" {
			if tp, ok := n.Type.(*ast.Ident); ok && tp.Name == w.oldident {
				tp.Name = w.newident
				*w.changed = true
			}
		} else {
			if tps, ok := n.Type.(*ast.SelectorExpr); ok {
				if tp, ok := tps.X.(*ast.Ident); ok && tp.Name == w.pkgname {
					if tps.Sel.Name == w.oldident {
						tps.Sel.Name = w.newident
						*w.changed = true
					}
				}
			}
		}
	case *ast.CallExpr:
		if w.pkgname == "" {
			if tp, ok := n.Fun.(*ast.Ident); ok && tp.Name == w.oldident {
				tp.Name = w.newident
				*w.changed = true
			}
		} else {
			if tps, ok := n.Fun.(*ast.SelectorExpr); ok {
				if tp, ok := tps.X.(*ast.Ident); ok && tp.Name == w.pkgname {
					if tps.Sel.Name == w.oldident {
						tps.Sel.Name = w.newident
						*w.changed = true
					}
				}
			}
		}
	}
	
	return w
}