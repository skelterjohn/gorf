package main

import (
	"os"
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
)

func ChangeIdent(kind, target, pkgname, oldname, newname string) (err os.Error) {
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
			switch kind {
			case "func":
				cli := ChangeLocalFuncWalker{
					oldident:oldname, newident:newname,
					changed:new(bool), exists:new(bool),
				}
				ast.Walk(&cli, ft)
				if *cli.exists {
					err = os.NewError(fmt.Sprintf("%s %s already exists in package %s in '%s'", kind, newname, pkgname, target))
				}
				if *cli.changed {
					changed = true
					RewriteSource(fpath, ft)
				}
			case "var":
				cli := ChangeLocalVarWalker{
					oldident:oldname, newident:newname,
					tok:token.VAR,
					changed:new(bool), exists:new(bool),
				}
				ast.Walk(&cli, ft)
				if *cli.exists {
					err = os.NewError(fmt.Sprintf("%s %s already exists in package %s in '%s'", kind, newname, pkgname, target))
				}
				if *cli.changed {
					changed = true
					RewriteSource(fpath, ft)
				}
			case "const":
				cli := ChangeLocalVarWalker{
					oldident:oldname, newident:newname,
					tok:token.CONST,
					changed:new(bool), exists:new(bool),
				}
				ast.Walk(&cli, ft)
				if *cli.exists {
					err = os.NewError(fmt.Sprintf("%s %s already exists in package %s in '%s'", kind, newname, pkgname, target))
				}
				if *cli.changed {
					changed = true
					RewriteSource(fpath, ft)
				}
			}
		} else {
			panic("never happens")
		}
	}
	
	if changed {
		for fpath, ft := range AllSources {
			ciw := CheckImportWalker{target:target}
			ast.Walk(&ciw, ft)
			if ciw.used {
				if ciw.pkgname == "" {
					ciw.pkgname = pkgname
				}
				crw := ChangeRemoteIdentWalker {
					target:target,
					pkgname:ciw.pkgname,
					oldident: oldname, newident:newname,
					changed:new(bool),
				}
				
				ast.Walk(&crw, ft)
				if *crw.changed {
					RewriteSource(fpath, ft)
				}
			
			}
		}
	} else {
		err = os.NewError(fmt.Sprintf("Found no occurences of %s in package %s in '%s'", oldname, pkgname, target))
	}
	
	return
}

type CheckImportWalker struct {
	target, pkgname string
	used bool
}

func (w *CheckImportWalker) Visit(node ast.Node) (v ast.Visitor) {
	switch n := node.(type) {
	case *ast.ImportSpec:
		if string(n.Path.Value) == QuoteTarget(w.target) {
			w.used = true
			if n.Name != nil {
				w.pkgname = n.Name.Name
			}
			return nil
		}
	}
	return w
}

type ChangeRemoteIdentWalker struct {
	target, pkgname string
	oldident, newident string
	changed *bool
	redefined bool	
}

func (w *ChangeRemoteIdentWalker) Visit(node ast.Node) (v ast.Visitor) {

	if w.redefined {
		return nil
	}
	
	if w.pkgname != "" && NodeRedefinesIdent(node, w.pkgname, w) {
		w.redefined = true
		return nil
	}
	if w.pkgname == "" && NodeRedefinesIdent(node, w.oldident, w) {
		w.redefined = true
		return nil
	}
	
	switch n := node.(type) {
	case *ast.BlockStmt:
		changer := &ChangeRemoteIdentWalker{}
		*changer = *w
		return changer
	case *ast.FuncDecl:
		changer := &ChangeRemoteIdentWalker{}
		*changer = *w
		return changer
	case *ast.SelectorExpr:
		if w.pkgname != "" {
			if id, ok := n.X.(*ast.Ident); ok && id.Name == w.pkgname {
				if n.Sel.Name == w.oldident {
					n.Sel.Name = w.newident
					*w.changed = true
				}
			}
		}
	case *ast.Ident:
		if w.pkgname == "" {
			if n.Name == w.oldident {
				n.Name = w.newident
				*w.changed = true
			}
		}
	}
	
	return w
}
