package main

import (
	"os"
	"fmt"
	"go/ast"
	"path/filepath"
)

func ChangePackages(target, oldpkg, newpkg string) (err os.Error) {
	if !IsLegalIdentifier(newpkg) {
		err = os.NewError(fmt.Sprintf("Package name %s is not a legal identifier", newpkg))
		return
	}

	ScanForTargets()
	
	if _, ok := GetDirTargets(target)[newpkg]; ok {
		err = os.NewError(fmt.Sprintf("Package %s already exists in '%s'", newpkg, target))
		return
	}
	
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
	if NodeRedefinesIdent(node, w.oldpkg, w) {
		w.redefined = true
		return nil
	}
	
	switch n := node.(type) {
	case *ast.BlockStmt:
		changer := &PackageIdentChanger{}
		*changer = *w
		return changer
	case *ast.FuncDecl:
		changer := &PackageIdentChanger{}
		*changer = *w
		return changer
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
	name string
	needsChange bool
}

func (w *ImportPathChecker) Visit(node ast.Node) (v ast.Visitor) {
	switch n := node.(type) {
	case *ast.ImportSpec:
		if string(n.Path.Value) == QuoteTarget(w.path) {
			if n.Name == nil {	
				w.needsChange = true
			}
			return nil
		}
	}
	return w
}