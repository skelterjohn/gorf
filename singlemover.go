package main

import (
	"fmt"
	"os"
	"unicode"
	"utf8"
	"path/filepath"
	"go/token"
	"go/printer"
	"go/ast"
	"rog-go.googlecode.com/hg/exp/go/types"
)

func MoveSingle(oldpath, newpath string, names []string) (err os.Error) {
	for _, name := range names {
		if !IsLegalIdentifier(name) {
			return MakeErr("Name %s is not a legal identifier", name)
		}
	}
	
	var sm SingleMover
	
	pkg := LocalImporter(oldpath)
	sm.pkg = pkg
	sm.oldpath = oldpath
	sm.newpath = newpath
	
	moveNodes := make(map[ast.Node]*ast.Object)
	sm.moveNodes = moveNodes
	moveObjs := make(map[*ast.Object]ast.Node)
	sm.moveObjs = moveObjs
	allObjs := make(AllDeclFinder)
	sm.allObjs = allObjs
	
	//get all top level decl objs
	ast.Walk(allObjs, pkg)
	
	//find the nodes we want to move
	for _, name := range names {
		if !sm.CollectNameObjs(name) {
			return MakeErr("Unable to find %s in '%s'", name, oldpath)
		}
	}
	for node, obj := range moveNodes {
		moveObjs[obj] = node
	}

	//the objs in remainingObjs are not being moved to the new package
	remainingObjs := make(map[*ast.Object]bool)
	sm.remainingObjs = remainingObjs
	for obj := range sm.allObjs {
		if _, ok := sm.moveObjs[obj]; !ok {
			sm.remainingObjs[obj] = true
		}
	}
	
	//get a list of objects that are unexported (and therefore if they
	//are referenced elsewhere, the move cannot happen)
	sm.unexportedObjs = make(map[*ast.Object]bool)
	sm.CollectUnexportedObjs()
	
	err = sm.CreateNewSource()
	if err != nil {
		return
	}
	
	//identify locations in pkg source files that need to now change
	err = sm.RemoveUpdatePkg()
	if err != nil {
		return
	}
	
	//make changes in packages that import this one
	err = sm.UpdateOther()
	if err != nil {
		return
	}
	return
}


type SingleMover struct {
	pkg *ast.Package
	oldpath, newpath string
	moveNodes map[ast.Node]*ast.Object
	moveObjs map[*ast.Object]ast.Node
	allObjs map[*ast.Object]ast.Node
	remainingObjs map[*ast.Object]bool
	unexportedObjs map[*ast.Object]bool
	referenceBack bool
}

func (this *SingleMover) CollectNameObjs(name string) (found bool) {
	for _, file := range this.pkg.Files {
		fdl := DeclFinder{oldname:name}
		ast.Walk(&fdl, file)
		if fdl.Obj != nil {
			found = true
			
			this.moveNodes[fdl.Node] = fdl.Obj
			
			mf := MethodFinder {
				Receiver : fdl.Obj,
				NodeObjs : make(map[ast.Node]*ast.Object),
			}
			
			ast.Walk(&mf, this.pkg)
			
			for node, obj := range mf.NodeObjs {
				this.moveNodes[node] = obj
			}
		}
	}
	return
}

func (this *SingleMover) CollectUnexportedObjs() {
	this.unexportedObjs = make(map[*ast.Object]bool)
	for node, obj := range this.moveNodes {
		//printer.Fprint(os.Stdout, token.NewFileSet(), node)
		//fmt.Printf("\n%v %T\n", obj, node)
		
		if !unicode.IsUpper(utf8.NewString(obj.Name).At(0)) {
			this.unexportedObjs[obj] = true
		}
		
		if ts, ok := node.(*ast.TypeSpec); ok {
			if st, ok := ts.Type.(*ast.StructType); ok {
				for _, field := range st.Fields.List {
					for _, name := range field.Names {
						if !name.IsExported() {
							obj, _ := types.ExprType(name, LocalImporter)
							this.unexportedObjs[obj] = true
						}
					}
				} 
			}
		}
	}
	return
}

func (this *SingleMover) CreateNewSource() (err os.Error) {
	
	liw := make(ListImportWalker)
	for n := range this.moveNodes {
		ast.Walk(liw, n)
	}
	finalImports := make(map[*ast.ImportSpec]bool)
	for obj, is := range liw {
		if _, ok := this.moveObjs[obj]; !ok {
			finalImports[is] = true
		}
	}
	
	newfile := &ast.File {
		Name : &ast.Ident{ Name : this.pkg.Name },
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
	
	for mn := range this.moveNodes {
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
	for n := range this.moveNodes {
		ast.Walk(&npf, n)
	}
	
	var pkgfiles []*ast.File
	for _, pkgfile := range this.pkg.Files {
		pkgfiles = append(pkgfiles, pkgfile)
	}
	oldPkgNewName := GetUniqueIdent(pkgfiles, this.pkg.Name)
	
	needOldImport := false
	
	this.referenceBack = false
	
	for expr, parent := range npf.ExprParents {
		obj, _ := types.ExprType(expr, LocalImporter)
		if _, ok := this.moveObjs[obj]; ok {
			continue
		}
		
		if _, ok := this.allObjs[obj]; !ok {
			continue
		}
		
		if !unicode.IsUpper(utf8.NewString(obj.Name).At(0)) {
			position := AllSources.Position(expr.Pos())
			fmt.Printf("At %v ", position)
			printer.Fprint(os.Stdout, token.NewFileSet(), expr)
			fmt.Println()
			err = MakeErr("Can't move code that references unexported objects")
			return
		}
		
		needOldImport = true
		this.referenceBack = true
		
		getSel := func(idn *ast.Ident) *ast.SelectorExpr {
			return &ast.SelectorExpr {
				X : &ast.Ident {
					Name : oldPkgNewName,
					NamePos : idn.NamePos,
				},
				Sel : idn,
			}
		}
	
		switch p := parent.(type) {
		case *ast.CallExpr:
			if idn, ok := expr.(*ast.Ident); ok { 
				p.Fun = getSel(idn)
			} else {
				err = MakeErr("CallExpr w/ unexpected type %T\n", expr)
				return
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
			err = MakeErr("Unexpected parent %T\n", parent)
			return
		}
	}
	
	if needOldImport {
		is := &ast.ImportSpec {
			Name : &ast.Ident{Name:oldPkgNewName},
			Path : &ast.BasicLit{Value:QuotePath(this.oldpath)},
		}
		gdl := &ast.GenDecl {
			Tok : token.IMPORT,
			Specs : []ast.Spec{is},
		}
		newfile.Decls = append([]ast.Decl{gdl}, newfile.Decls...)
	}
	
	err = os.MkdirAll(this.newpath, 0755)
	if err != nil {
		return
	}
	newSourcePath := filepath.Join(this.newpath, this.pkg.Name+".go")
	
	err = NewSource(newSourcePath, newfile)
	if err != nil {
		return
	}
	
	return
}

func (this *SingleMover) RemoveUpdatePkg() (err os.Error) {
	for fpath, file := range this.pkg.Files {
	
		urw := ReferenceWalker {
			UnexportedObjs : this.unexportedObjs,
			SkipNodes : this.moveNodes,
			MoveObjs : this.moveObjs,
			SkipNodeParents : make(map[ast.Node]ast.Node),
			GoodReferenceParents : make(map[ast.Node]ast.Node),
			BadReferences : new([]ast.Node),
		}
		ast.Walk(&urw, file)
		
		if len(*urw.BadReferences) != 0 {
			fmt.Printf("Cannot move some objects:\n")
			for node := range this.moveNodes {
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
			return MakeErr("Objects to be moved in '%s' contains unexported objects referenced elsewhere in the package", this.oldpath)
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
			for _, file := range this.pkg.Files {
				iuc := make(ImportUseCollector)
				ast.Walk(iuc, file)
				
				ast.Walk(ImportFilterWalker(iuc), file)
			}
		}
				
		//if this file refernces things that are moving, import the new package
		if len(urw.GoodReferenceParents) != 0 {
			if this.referenceBack {
				return MakeErr("Moving objects from %s would create a cycle", this.oldpath)
			}
		
			newpkgname := GetUniqueIdent([]*ast.File{file}, this.pkg.Name)
			
			//construct the import
			is := &ast.ImportSpec {
				Name : &ast.Ident{Name: newpkgname},
				Path : &ast.BasicLit{
					Kind : token.STRING,
					Value : QuotePath(this.newpath),
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
							NamePos : idn.NamePos,
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
	return
}

func (this *SingleMover) UpdateOther() (err os.Error) {
	for _, path := range ImportedBy[QuotePath(this.oldpath)] {
		opkg := LocalImporter(path)
		
		for fpath, file := range opkg.Files {
			rw := ReferenceWalker {
				UnexportedObjs : make(map[*ast.Object]bool),
				MoveObjs : this.moveObjs,
				SkipNodes : make(map[ast.Node]*ast.Object),
				SkipNodeParents : make(map[ast.Node]ast.Node),
				GoodReferenceParents : make(map[ast.Node]ast.Node),
				BadReferences : &[]ast.Node{},
			}
			ast.Walk(&rw, file)
			
			if len(rw.GoodReferenceParents) == 0 {
				continue
			}
			
			newpkgname := GetUniqueIdent([]*ast.File{file}, this.pkg.Name)
			
			//construct the import
			nis := &ast.ImportSpec {
				Name : &ast.Ident{Name: newpkgname},
				Path : &ast.BasicLit{
					Kind : token.STRING,
					Value : QuotePath(this.newpath),
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
								NamePos : sel.X.Pos(),
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
				Objs : this.remainingObjs,
			}
			ast.Walk(&oc, file)
			if !oc.Found {
				ast.Walk(&ImportRemover{nil, this.oldpath}, file)
			}
			
			err = RewriteSource(fpath, file)
			if err != nil {
				return
			}
		}
	}
	
	return
}
