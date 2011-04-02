// Copyright 2011 John Asmuth. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"unicode"
	"utf8"
	"os"
	"go/ast"
	"go/printer"
	"go/token"
	"path/filepath"
	"rog-go.googlecode.com/hg/exp/go/types"
)

func MoveCmd(args []string) (err os.Error) {
	if len(args) < 2 {
		return MakeErr("Usage: gorf [flags] move <old path> <new path> [<name>+]")
	}
	
	oldpath, newpath := args[0], args[1]
	
	if oldpath == newpath {
		return MakeErr("Old path and new path are the same")
	}
	
	err = ScanAllForImports(LocalRoot)
	if err != nil {
		return
	}
	
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

func MoveSingle(oldpath, newpath string, names []string) (err os.Error) {
	for _, name := range names {
		if !IsLegalIdentifier(name) {
			return MakeErr("Name %s is not a legal identifier", name)
		}
	}
	
	pkg := LocalImporter(oldpath)
	
	unexportedObjs := make(map[*ast.Object]bool)
	moveNodes := make(map[ast.Node]*ast.Object)
	moveObjs := make(map[*ast.Object]ast.Node)
	allObjs := make(AllDeclFinder)
	
	ast.Walk(allObjs, pkg)
	
	//find the nodes we want to move
	for _, name := range names {
		found := false
		for _, file := range pkg.Files {
			fdl := DeclFinder{oldname:name}
			ast.Walk(&fdl, file)
			if fdl.Obj != nil {
				found = true
			
				moveObjs[fdl.Obj] = fdl.Node
				moveNodes[fdl.Node] = fdl.Obj
				
				mf := MethodFinder {
					Receiver : fdl.Obj,
					NodeObjs : make(map[ast.Node]*ast.Object),
				}
				
				ast.Walk(&mf, pkg)
				
				for node, obj := range mf.NodeObjs {
					moveObjs[obj] = node
					moveNodes[node] = obj
				}
			}
		}
		
		if !found {
			return MakeErr("Unable to find %s in '%s'", name, oldpath)
		}
	}

	remainingObjs := make(map[*ast.Object]bool)
	for obj := range allObjs {
		if _, ok := moveObjs[obj]; !ok {
			remainingObjs[obj] = true
		}
	}
	
	
	// make a list of objs that cannot be referenced from outside the moved nodes
	for node, obj := range moveNodes {
		//printer.Fprint(os.Stdout, token.NewFileSet(), node)
		//fmt.Printf("\n%v %T\n", obj, node)
		
		if !unicode.IsUpper(utf8.NewString(obj.Name).At(0)) {
			unexportedObjs[obj] = true
		}
		
		if ts, ok := node.(*ast.TypeSpec); ok {
			if st, ok := ts.Type.(*ast.StructType); ok {
				for _, field := range st.Fields.List {
					for _, name := range field.Names {
						if !name.IsExported() {
							obj, _ := types.ExprType(name, LocalImporter)
							unexportedObjs[obj] = true
						}
					}
				} 
			}
		}
	}
	
	//get the imports we need for the new source file
	liw := make(ListImportWalker)
	for n := range moveNodes {
		ast.Walk(liw, n)
	}
	finalImports := make(map[*ast.ImportSpec]bool)
	for obj, is := range liw {
		if _, ok := moveObjs[obj]; !ok {
			finalImports[is] = true
		}
	}
	
	//write the new source file
	newfile := &ast.File {
		Name : &ast.Ident{ Name : pkg.Name },
	}
	
	
	if len(finalImports) != 0 {
		for is := range finalImports {
			gdl := &ast.GenDecl {
				Tok : token.IMPORT,
				Specs : []ast.Spec{is},
			}
			newfile.Decls = append(newfile.Decls, gdl)
		}
	}
	
	for mn := range moveNodes {
		switch m := mn.(type) {
		case ast.Decl:
			newfile.Decls = append(newfile.Decls, m)
		case *ast.TypeSpec:
			gdl := &ast.GenDecl {
				Tok : token.TYPE,
				Specs : []ast.Spec{m},
			}
			newfile.Decls = append(newfile.Decls, gdl)
		}
		
	}
	
	npf := ExprParentFinder {
		ExprParents : make(map[ast.Expr]ast.Node),
	}
	for n := range moveNodes {
		ast.Walk(&npf, n)
	}
	
	var pkgfiles []*ast.File
	for _, pkgfile := range pkg.Files {
		pkgfiles = append(pkgfiles, pkgfile)
	}
	oldPkgNewName := GetUniqueIdent(pkgfiles, pkg.Name)
	
	needOldImport := false
	
	referenceBack := false
	
	for expr, parent := range npf.ExprParents {
		obj, _ := types.ExprType(expr, LocalImporter)
		if _, ok := moveObjs[obj]; ok {
			continue
		}
		
		if _, ok := allObjs[obj]; !ok {
			continue
		}
		
		if !unicode.IsUpper(utf8.NewString(obj.Name).At(0)) {
			position := AllSources.Position(expr.Pos())
			fmt.Printf("At %v ", position)
			printer.Fprint(os.Stdout, token.NewFileSet(), expr)
			fmt.Println()
			return MakeErr("Can't move code that references unexported objects")
		}
		
		needOldImport = true
		referenceBack = true
		
		getSel := func(idn *ast.Ident) *ast.SelectorExpr {
			return &ast.SelectorExpr {
				X : &ast.Ident {
					Name : oldPkgNewName,
				},
				Sel : idn,
			}
		}
	
		switch p := parent.(type) {
		case *ast.CallExpr:
			if idn, ok := expr.(*ast.Ident); ok { 
				p.Fun = getSel(idn)
			} else {
				return MakeErr("CallExpr w/ unexpected type %T\n", expr)
			}
		case *ast.AssignStmt:
			for i, x := range p.Lhs {
				if x == expr {
					if idn, ok := x.(*ast.Ident); ok {
						p.Lhs[i] = getSel(idn)
					}
				}
			}
			for i, x := range p.Rhs {
				if x == expr {
					if idn, ok := x.(*ast.Ident); ok {
						p.Rhs[i] = getSel(idn)
					}
				}
			}
		default:
			return MakeErr("Unexpected parent %T\n", parent)
		}
	}
	
	if needOldImport {
		is := &ast.ImportSpec {
			Name : &ast.Ident{Name:oldPkgNewName},
			Path : &ast.BasicLit{Value:QuotePath(oldpath)},
		}
		gdl := &ast.GenDecl {
			Tok : token.IMPORT,
			Specs : []ast.Spec{is},
		}
		newfile.Decls = append([]ast.Decl{gdl}, newfile.Decls...)
	}
	
	err = os.MkdirAll(newpath, 0755)
	if err != nil {
		return
	}
	newSourcePath := filepath.Join(newpath, pkg.Name+".go")
	
	err = NewSource(newSourcePath, newfile)
	if err != nil {
		return
	}
	
	//identify locations in other pkg source files that need to now change
	for fpath, file := range pkg.Files {
	
		urw := ReferenceWalker {
			UnexportedObjs : unexportedObjs,
			SkipNodes : moveNodes,
			MoveObjs : moveObjs,
			SkipNodeParents : make(map[ast.Node]ast.Node),
			GoodReferenceParents : make(map[ast.Node]ast.Node),
			BadReferences : new([]ast.Node),
		}
		ast.Walk(&urw, file)
		
		if len(*urw.BadReferences) != 0 {
			fmt.Printf("Cannot move %v:\n", names)
			for node := range moveNodes {
				printer.Fprint(os.Stdout, token.NewFileSet(), node)
				fmt.Println()
			}
			fmt.Println("Unexported objects referenced:")
			for _, node := range *urw.BadReferences {
				position := AllSources.Position(node.Pos())
				fmt.Printf("At %v ", position)
				printer.Fprint(os.Stdout, token.NewFileSet(), node)
				fmt.Println()
			}
			return MakeErr("%v in '%s' contains unexported objects referenced elsewhere in the package", names, oldpath)
		}
		
		removedStuff := false
		
		// remove the old definitions
		for node, parent := range urw.SkipNodeParents {
			removedStuff = true
			//fmt.Printf("%T %v\n", parent, parent)
			
			switch pn := parent.(type) {
			case *ast.File:
				for i, n := range pn.Decls {
					if n == node {
						if len(pn.Decls) > 1 {
							pn.Decls[i], pn.Decls[len(pn.Decls)-1] = pn.Decls[len(pn.Decls)-1], pn.Decls[i]
						}
						pn.Decls = pn.Decls[:len(pn.Decls)-1]
						break
					}
				}
			case *ast.GenDecl:
				for i, n := range pn.Specs {
					if n == node {
						if pn.Lparen == 0 {
							pn.Lparen = n.Pos()
							pn.Rparen = n.End()
						}
						if len(pn.Specs) > 1 {
							pn.Specs[i], pn.Specs[len(pn.Specs)-1] = pn.Specs[len(pn.Specs)-1], pn.Specs[i]
						}
						pn.Specs = pn.Specs[:len(pn.Specs)-1]
						break
					}
				}
			default:
				return MakeErr("Unanticipated parent type: %T", pn)
			}	
		}
		
		//strip out imports that are unnecessary because things are no longer here
		if removedStuff {
			for _, file := range pkg.Files {
				iuc := make(ImportUseCollector)
				ast.Walk(iuc, file)
				
				ast.Walk(ImportFilterWalker(iuc), file)
			}
		}
				
		//if this file refernces things that are moving, import the new package
		if len(urw.GoodReferenceParents) != 0 {
			if referenceBack {
				return MakeErr("Moving %v from %s would create a cycle", names, oldpath)
			}
		
			newpkgname := GetUniqueIdent([]*ast.File{file}, pkg.Name)
			
			//construct the import
			is := &ast.ImportSpec {
				Name : &ast.Ident{Name: newpkgname},
				Path : &ast.BasicLit{
					Kind : token.STRING,
					Value : QuotePath(newpath),
				},
			}
			
			gd := &ast.GenDecl {
				Tok : token.IMPORT,
				Specs : []ast.Spec{is},
			}
			
			//stick it in there
			file.Decls = append([]ast.Decl{gd}, file.Decls...)
			
		
			//change the old references to talk about the new package, using our unique name
			for node, parent := range urw.GoodReferenceParents {
				getSel := func(idn *ast.Ident) *ast.SelectorExpr {
					return &ast.SelectorExpr {
						X : &ast.Ident {
							Name : newpkgname,
						},
						Sel : idn,
					}
				}
				
				switch p := parent.(type) {
				case *ast.CallExpr:
					if idn, ok := node.(*ast.Ident); ok { 
						p.Fun = getSel(idn)
					} else {
						
						return MakeErr("CallExpr w/ unexpected type %T\n", node)
					}
				case *ast.AssignStmt:
					for i, x := range p.Lhs {
						if x == node {
							if idn, ok := x.(*ast.Ident); ok {
								p.Lhs[i] = getSel(idn)
							}
						}
					}
					for i, x := range p.Rhs {
						if x == node {
							if idn, ok := x.(*ast.Ident); ok {
								p.Rhs[i] = getSel(idn)
							}
						}
					}
				case *ast.StarExpr:
					if p.X == node {
						if idn, ok := p.X.(*ast.Ident); ok {
							p.X = getSel(idn)
						}
					}
				default:
					return MakeErr("Unexpected local parent %T\n", parent)
				}
			}
		}
		
		if removedStuff {
			err = RewriteSource(fpath, file)
			if err != nil {
				return
			}
		}
	
	}
	
	//make changes in packages that import this one
	for _, path := range ImportedBy[QuotePath(oldpath)] {
		opkg := LocalImporter(path)
		
		for fpath, file := range opkg.Files {
			rw := ReferenceWalker {
				UnexportedObjs : make(map[*ast.Object]bool),
				MoveObjs : moveObjs,
				SkipNodes : make(map[ast.Node]*ast.Object),
				SkipNodeParents : make(map[ast.Node]ast.Node),
				GoodReferenceParents : make(map[ast.Node]ast.Node),
				BadReferences : &[]ast.Node{},
			}
			ast.Walk(&rw, file)
			
			if len(rw.GoodReferenceParents) == 0 {
				continue
			}
			
			newpkgname := GetUniqueIdent([]*ast.File{file}, pkg.Name)
			
			//construct the import
			nis := &ast.ImportSpec {
				Name : &ast.Ident{Name: newpkgname},
				Path : &ast.BasicLit{
					Kind : token.STRING,
					Value : QuotePath(newpath),
				},
			}
			
			ngd := &ast.GenDecl {
				Tok : token.IMPORT,
				Specs : []ast.Spec{nis},
			}
			file.Decls = append([]ast.Decl{ngd}, file.Decls...)
			
			for node, parent := range rw.GoodReferenceParents {
				getSel := func(sel *ast.SelectorExpr) *ast.SelectorExpr {
					obj, _ := types.ExprType(sel.X, LocalImporter)
					if obj.Kind == ast.Pkg {
						return &ast.SelectorExpr {
							X : &ast.Ident {
								Name : newpkgname,
							},
							Sel : sel.Sel,
						}
					}
					return sel
				}
				
				switch p := parent.(type) {
				case *ast.CallExpr:
					if sel, ok := node.(*ast.SelectorExpr); ok {
						
						p.Fun = getSel(sel)
					} else {
						
						return MakeErr("CallExpr w/ unexpected type %T\n", node)
					}
				case *ast.AssignStmt:
					for i, x := range p.Lhs {
						if x == node {
							if sel, ok := x.(*ast.SelectorExpr); ok {
								p.Lhs[i] = getSel(sel)
							}
						}
					}
					for i, x := range p.Rhs {
						if x == node {
							if sel, ok := x.(*ast.SelectorExpr); ok {
								p.Rhs[i] = getSel(sel)
							}
						}
					}
				case *ast.ValueSpec:
					if node == p.Type {
						if sel, ok := p.Type.(*ast.SelectorExpr); ok {
							p.Type = getSel(sel)
						}
					}
					for i, x := range p.Values {
						if x == node {
							if sel, ok := x.(*ast.SelectorExpr); ok {
								p.Values[i] = getSel(sel)
							}
						}
					}
				case *ast.StarExpr:
					if p.X == node {
						if sel, ok := p.X.(*ast.SelectorExpr); ok {
							p.X = getSel(sel)
						}
					}
				default:
					printer.Fprint(os.Stdout, AllSources, parent)
					return MakeErr("Unexpected remote parent %T\n", parent)
				}
			}
			
			//now that we've renamed some references, do we still need to import oldpath?
			oc := ObjChecker {
				Objs : remainingObjs,
			}
			ast.Walk(&oc, file)
			if !oc.Found {
				ast.Walk(&ImportRemover{nil, oldpath}, file)
			}
			
			err = RewriteSource(fpath, file)
			if err != nil {
				return
			}
		}
	}
	
	//return MakeErr("jk")
	
	/*
	for _, file := range pkg.Files {
		printer.Fprint(os.Stdout, AllSources, file)
		fmt.Println()
	}
	*/
	
	
	return
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
