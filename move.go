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
	if len(args) != 3 && len(args) != 2 {
		return MakeErr("Usage: gorf [flags] move <old path> <new path> [<name>]")
	}
	
	oldpath, newpath := args[0], args[1]
	
	if oldpath == newpath {
		return MakeErr("Old path and new path are the same")
	}
	
	ScanAllForImports(LocalRoot)
	
	defer func() {
		if err != nil {
			 UndoCmd([]string{})
		}
	}()
	
	if PackageTops[newpath] != nil {
		return MakeErr("New path %s already has a package (did you mean to merge?)", newpath)
	}
	
	pkg := LocalImporter(oldpath)
	
	if pkg == nil {
		return MakeErr("Old path %s has no package", oldpath)
	}

	if len(args) == 3 {
		name := args[2]
		err = MoveSingle(oldpath, newpath, name)
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

func MoveSingle(oldpath, newpath, name string) (err os.Error) {
	if !IsLegalIdentifier(name) {
		return MakeErr("Name %s is not a legal identifier", name)
	}
	
	pkg := LocalImporter(oldpath)
	
	unexportedObjs := make(map[*ast.Object]bool)
	moveNodes := make(map[ast.Node]*ast.Object)
	moveObjs := make(map[*ast.Object]ast.Node)
	allObjs := make(AllDeclFinder)
	
	ast.Walk(allObjs, pkg)
	
	for _, file := range pkg.Files {
		fdl := DeclFinder{oldname:name}
		ast.Walk(&fdl, file)
		if fdl.Obj != nil {
			moveObjs[fdl.Obj] = fdl.Node
			moveNodes[fdl.Node] = fdl.Obj
		}
	}
	
	
	
	if len(moveNodes) == 0 {
		return MakeErr("Unable to find %s in '%s'", name, oldpath)
	}
	
	//find the nodes we want to move
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
		if md, ok := mn.(ast.Decl); ok {
			newfile.Decls = append(newfile.Decls, md)
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
	
	/*
	fmt.Println("unexported:")
	for ueo, _ := range unexportedObjs {
		fmt.Printf("%v\n", ueo)
	}
	*/

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
			fmt.Printf("Cannot move %s:\n", name)
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
			return MakeErr("%s in '%s' contains unexported objects referenced elsewhere in the package", name, oldpath)
		}
		
		// remove the old definitions
		for node, parent := range urw.SkipNodeParents {
			
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
		
		//if this file refernces things that are moving, import the new package
		if len(urw.GoodReferenceParents) != 0 {
			if referenceBack {
				return MakeErr("Moving %s from %s would create a cycle", name, oldpath)
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
				default:
					return MakeErr("Unexpected parent %T\n", parent)
				}
			}
			
			
			err = RewriteSource(fpath, file)
			if err != nil {
				return
			}
		}
	
	}
	
	//make changes in packages that import this one
	
			
			//stick it in there
	
	
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
					return &ast.SelectorExpr {
						X : &ast.Ident {
							Name : newpkgname,
						},
						Sel : sel.Sel,
					}
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
				default:
					return MakeErr("Unexpected parent %T\n", parent)
				}
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
