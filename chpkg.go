package main

import (
	"os"
	"fmt"
	"go/ast"
	"go/token"
	"path/filepath"
)

func ChangePackages(target, oldpkg, newpkg string) (err os.Error) {
	ScanForTargets()
	
	dirTarget, ok := GetDirTargets(target)[oldpkg]
	if !ok {
		err = os.NewError(fmt.Sprintf("No target %s found in '%s'", oldpkg, target))
		return
	}
	for _, src := range dirTarget.Source {
		BackupSource(filepath.Join(dirTarget.Path, src))
		err = ChangePackage(filepath.Join(dirTarget.Path, src), oldpkg, newpkg)
		if err != nil {
			return
		}
	}
	
	for src, ft := range AllSources {
		err = ChangePackageRef(src, ft, target, oldpkg, newpkg)
		if err != nil {
			return
		}
	}
	
	return
}

func ChangePackage(src, oldpkg, newpkg string) (err os.Error) {
	ft := AllSources[src]
	
	if ft.Name.Name != oldpkg {
		panic("never happens")
	}
	ft.Name.Name = newpkg

	err = RewriteSource(src, ft)
	
	return
}

func ChangePackageRef(src string, ft *ast.File, target, oldpkg, newpkg string) (err os.Error) {
	target = "\""+target+"\""
	checker := &ImportPathChecker{path:target}
	ast.Walk(checker, ft)
	if !checker.needsChange {
		//fmt.Println(src, "doesn't need it")
		//this file doesn't import our target, or it names it explicitly
		return
	}
	
	changer := &PackageIdentChanger{oldpkg:oldpkg, newpkg:newpkg, changed:new(bool)}
	ast.Walk(changer, ft)
	if *changer.changed {
		err = RewriteSource(src, ft)
	}
	
	return
}

type PackageIdentChanger struct {
	oldpkg, newpkg string
	redefined bool
	changed *bool
}

func (w *PackageIdentChanger) Visit(node ast.Node) (v ast.Visitor) {
	if w.redefined {
		return nil
	}
	/*
	if node != nil {
		fmt.Printf("Node type: %T\nNode: %v\n\n", node, node)
	}
	*/
	switch n := node.(type) {
	case *ast.BlockStmt:
		changer := &PackageIdentChanger{oldpkg:w.oldpkg, newpkg:w.newpkg, changed:w.changed}
		return changer
	case *ast.AssignStmt:
		if n.Tok == token.DEFINE {
			for _, e := range n.Lhs {
				if id, ok := e.(*ast.Ident); ok {
					if id.Name == w.oldpkg {
						w.redefined = true
						return nil
					}
				}
			}
		}
	case *ast.GenDecl:
		for _, spec := range n.Specs {
			if vs, ok := spec.(*ast.ValueSpec); ok {
				for _, id := range vs.Names {
					if id.Name == w.oldpkg {
						w.redefined = true
						return nil
					}
				}
			}
		}
	case *ast.SelectorExpr:
		if id, ok := n.X.(*ast.Ident); ok {
			if id.Name == w.oldpkg {
				id.Name = w.newpkg
				*w.changed = true
			}
		}
	}
	return w
}

type ImportPathChecker struct {
	path string
	needsChange bool
}

func (w *ImportPathChecker) Visit(node ast.Node) (v ast.Visitor) {
	switch n := node.(type) {
	case *ast.ImportSpec:
		if string(n.Path.Value) == w.path {
			if n.Name == nil {	
				w.needsChange = true
			}
			return nil
		}
	}
	return w
}