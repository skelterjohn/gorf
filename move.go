// Copyright 2011 John Asmuth. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"strings"
	"os"
	"go/ast"
	"path/filepath"
	"rog-go.googlecode.com/hg/exp/go/types"
)

func MoveAllCmd(args []string) (err os.Error) {
	if len(args) != 2 {
		return MakeErr("Usage: gorf [flags] moveall <old path> <new path>")
	}
	
	oldPath, newPath := filepath.Clean(args[0]), filepath.Clean(args[1])
	
	err = ScanAllForImports(LocalRoot)
	if err != nil {
		return
	}
	
	prefix := fmt.Sprintf("%s%c", oldPath, filepath.Separator)
	fmt.Println("prefix:", prefix)
	for opath := range PackageTops {
		opath = fmt.Sprintf("%s%c", filepath.Clean(opath), filepath.Separator)
		if strings.HasPrefix(opath, prefix) {
			tail := opath[len(prefix):]
			npath := filepath.Join(newPath, tail)
			err = MoveCmd([]string{opath, npath})
			if err != nil {
				return
			}
		}
	}
	
	return
}

func MoveCmd(args []string) (err os.Error) {
	if len(args) < 2 {
		return MakeErr("Usage: gorf [flags] move <old path> <new path> [<name>+]")
	}
	
	oldpath, newpath := filepath.Clean(args[0]), filepath.Clean(args[1])
	
	if oldpath == newpath {
		return MakeErr("Old path and new path are the same")
	}
	
	err = ScanAllForImports(LocalRoot)
	if err != nil {
		return
	}
	
	PreloadImportedBy(oldpath)
	
	defer func() {
		if err != nil {
			 UndoCmd([]string{})
		}
	}()
	
	if PackageTops[oldpath] == nil {
		return MakeErr("Old path %s has no local package", oldpath)
	}
	
	if PackageTops[newpath] != nil {
		return MakeErr("New path %s already has a package (did you mean to merge?)", newpath)
	}
	
	pkg := LocalImporter(oldpath)

	if len(args) >= 3 {
		err = MoveSingle(oldpath, newpath, args[2:])
		return
	}
	
	os.MkdirAll(filepath.Join(LocalRoot, newpath), 0755)
	for fpath := range pkg.Files {
		_, base := filepath.Split(fpath)
		npath := filepath.Join(LocalRoot, newpath, base)
		err = MoveSource(fpath, npath)
		if err != nil {
			return
		}
	}
	
	for _, ip := range ImportedBy[QuotePath(oldpath)] {
		ipkg := LocalImporter(ip)
		for fpath, file := range ipkg.Files {
			pcw := PathChangeWalker{OldPath:oldpath, NewPath:newpath}
			ast.Walk(&pcw, file)
			if pcw.Updated {
				err = RewriteSource(fpath, file)
				if err != nil {
					return
				}
			}
		}
	}
	
	return
}

type PathChangeWalker struct {
	OldPath, NewPath string
	Updated bool
}

func (this *PathChangeWalker) Visit(node ast.Node) ast.Visitor {
	if this.Updated {
		return nil
	}
	if n, ok := node.(*ast.ImportSpec); ok {
		if string(n.Path.Value) == QuotePath(this.OldPath) {
			n.Path.Value = QuotePath(this.NewPath)
			this.Updated = true
			return nil
		}
	}
	return this
}

type ImportRemover struct {
	Parent ast.Node
	Path string
}

func (this *ImportRemover) Visit(node ast.Node) ast.Visitor {
	if is, ok := node.(*ast.ImportSpec); ok {
		if is.Path.Value == QuotePath(this.Path) {
			switch p := this.Parent.(type) {
			case *ast.GenDecl:
				for i, gis := range p.Specs {
					if gis == is {
						l := len(p.Specs)
						if l > 1 {
							p.Specs[i], p.Specs[l-1] = p.Specs[l-1], p.Specs[i]
						} else if p.Lparen == 0 {
							p.Lparen = is.Pos()
							p.Rparen = is.Pos()
						}
						p.Specs = p.Specs[:l-1]
					}
				}
			}
			return nil
		}
	}
	
	return &ImportRemover {
		Parent : node,
		Path : this.Path,
	}
}

type ImportUseCollector map[*ast.ImportSpec]bool

func (this ImportUseCollector) Visit(node ast.Node) ast.Visitor {
	if _, ok := node.(*ast.ImportSpec); ok {
		return nil
	}

	if expr, ok := node.(ast.Expr); ok {
		_, typ := types.ExprType(expr, LocalImporter)
		if typ.Node != node {
			if is, ok2 := typ.Node.(*ast.ImportSpec); ok2 {
				this[is] = true
			}
		}
	}
	
	return this
}

type ImportFilterWalker ImportUseCollector
func (this ImportFilterWalker) Visit(node ast.Node) ast.Visitor {
	if gdl, ok := node.(*ast.GenDecl); ok {
		var newspecs []ast.Spec 
		for _, spec := range gdl.Specs {
			if is, ok2 := spec.(*ast.ImportSpec); ok2 {
				if !this[is] {
					continue
				}
			}
			newspecs = append(newspecs, spec)
		}
		gdl.Specs = newspecs
	}
	
	if _, ok := node.(*ast.BlockStmt); ok {
		return nil
	}
	
	return this
}

type ObjChecker struct {
	Objs map[*ast.Object]bool
	Found bool
}

func (this *ObjChecker) Visit(node ast.Node) ast.Visitor {
	if this.Found {
		return nil
	}
	if expr, ok := node.(ast.Expr); ok {
		obj, _ := types.ExprType(expr, LocalImporter)
		if this.Objs[obj] {
			this.Found = true
			return nil
		}
	}
	return this
}

type ReferenceWalker struct {
	Parent ast.Node
	UnexportedObjs map[*ast.Object]bool
	MoveObjs map[*ast.Object]ast.Node
	SkipNodes map[ast.Node]*ast.Object
	SkipNodeParents map[ast.Node]ast.Node
	GoodReferenceParents map[ast.Node]ast.Node
	BadReferences *[]ast.Node
}

func (this *ReferenceWalker) Visit(node ast.Node) ast.Visitor {
	if _, ok := this.SkipNodes[node]; ok {
		this.SkipNodeParents[node] = this.Parent
		return nil
	}
	
	if expr, ok := node.(ast.Expr); ok {
		obj, _ := types.ExprType(expr, LocalImporter)
		if this.UnexportedObjs[obj] {
			*this.BadReferences = append(*this.BadReferences, node)
		} else if _, ok2 := this.MoveObjs[obj]; ok2 {
			this.GoodReferenceParents[node] = this.Parent
		}
	}
	
	next := new(ReferenceWalker)
	*next = *this
	
	next.Parent = node
	
	return next
}

type ListImportWalker map[*ast.Object]*ast.ImportSpec

func (this ListImportWalker) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.SelectorExpr:
		ast.Walk(this, n.X)
		//skip n.Sel, we don't need to import for it
		return nil
	case *ast.Ident:
		obj, typ := types.ExprType(n, LocalImporter)
		if is, ok := typ.Node.(*ast.ImportSpec); ok {
			this[obj] = is
		}
	}
	
	return this
}

type ExprParentFinder struct {
	Parent ast.Node
	ExprParents map[ast.Expr]ast.Node
}

func (this *ExprParentFinder) Visit(node ast.Node) ast.Visitor {
	if ex, ok := node.(ast.Expr); ok {
		this.ExprParents[ex] = this.Parent
	}
	 
	return &ExprParentFinder{
		Parent : node,
		ExprParents : this.ExprParents,
	}
}

type AllDeclFinder map[*ast.Object]ast.Node

func (this AllDeclFinder) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.BlockStmt:
		return nil
	case *ast.ValueSpec:
		for _, name := range n.Names {
			obj, _ := types.ExprType(name, LocalImporter)
			this[obj] = node
		}
		return nil
	case *ast.FuncDecl:
		obj, _ := types.ExprType(n.Name, LocalImporter)
		this[obj] = node
		return nil
	case *ast.TypeSpec:
		obj, _ := types.ExprType(n.Name, LocalImporter)
		this[obj] = node
		return nil
	}
	return this
}

type MethodFinder struct {
	Receiver *ast.Object
	NodeObjs map[ast.Node]*ast.Object
}

func (this *MethodFinder) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.BlockStmt:
		return nil
	case *ast.FuncDecl:
		if n.Recv != nil {
			for _, field := range n.Recv.List {
				expr := field.Type
				if se, ok := expr.(*ast.StarExpr); ok {
					expr = se.X
				}
				obj, _ := types.ExprType(expr, LocalImporter)
				if obj == this.Receiver {
					fobj, _ := types.ExprType(n.Name, LocalImporter)
					this.NodeObjs[n] = fobj
				}
			}
		}
		return nil
	}
	return this
}
