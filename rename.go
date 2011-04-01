// Copyright 2011 John Asmuth. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"go/ast"
	"strings"
	"rog-go.googlecode.com/hg/exp/go/types"
)	

func RenameCmd(args []string) (err os.Error) {
	if len(args) != 3 {
		return MakeErr("Usage: gorf [flags] rename <path> [<type>.]<old name> <new name>")
	}
	path, oldname, newname := args[0], args[1], args[2]
	
	if oldnametoks := strings.Split(oldname, ".", 2); len(oldnametoks) == 2 {
		return FieldCmd([]string{path, oldnametoks[0], oldnametoks[1], newname})
	}
	
	if !IsLegalIdentifier(oldname) {
		return MakeErr("Old name %s is not a legal identifier", oldname)
	}
	if !IsLegalIdentifier(newname) {
		return MakeErr("New name %s is not a legal identifier", newname)
	}
	if oldname == newname {
		return MakeErr("Old name and new name are the same")
	}
	
	ScanAllForImports(LocalRoot)
	
	defer func() {
		if err != nil {
			 UndoCmd([]string{})
		}
	}()
	
	pkg := LocalImporter(path)
	
	if pkg == nil {
		return MakeErr("No package found in %s", path)
	}
	
	updated := false
	
	var Obj *ast.Object
	
	for fpath, file := range pkg.Files {
		fdl := DeclFinder{oldname:oldname, newname:newname}
		ast.Walk(&fdl, file)
		
		if fdl.NameExists {
			return MakeErr("Name %s already exists", newname)
		}
		Obj = fdl.Obj
		
		if Obj != nil {
			updated = true	
		}
	
		if updated {
			RenameInFile(file, newname, Obj)
			RewriteSource(fpath, file)
		}
	}
	
	if updated {
		err = RenameInAll(path, newname, Obj)
	}
	
	return
}

type DeclFinder struct {
	oldname, newname string
	NameExists bool
	Obj *ast.Object
	Name *ast.Ident
}

func (this *DeclFinder) Visit(node ast.Node) ast.Visitor {
	if this.Obj != nil {
		return nil
	}
	switch n := node.(type) {
	case *ast.BlockStmt:
		return nil
	case *ast.ValueSpec:
		for _, name := range n.Names {
			if name.Name == this.newname {
				this.NameExists = true
				return nil
			}
			if name.Name == this.oldname {
				//this.Name = name
				this.Obj, _ = types.ExprType(name, LocalImporter)
				return nil
			}
		}
		return nil
	case *ast.FuncDecl:
		if n.Name.Name == this.newname {
			this.NameExists = true
		}
		if n.Name.Name == this.oldname {
			//this.Name = n.Name
			this.Obj, _ = types.ExprType(n.Name, LocalImporter)
		}
		return nil
	case *ast.TypeSpec:
		if n.Name.Name == this.newname {
			this.NameExists = true
		}
		if n.Name.Name == this.oldname {
			//this.Name = n.Name
			this.Obj, _ = types.ExprType(n.Name, LocalImporter)
		}
		return nil
	}
	return this
}


func RenameInAll(path string, newname string, Obj *ast.Object) (err os.Error) {
	for _, ip := range ImportedBy[QuotePath(path)] {
		ipkg := LocalImporter(ip)
		for fpath, file := range ipkg.Files {
			if RenameInFile(file, newname, Obj) {
				err = RewriteSource(fpath, file)
				if err != nil {
					return
				}
			}
		}
	}
	return
}

func RenameInFile(file *ast.File, NewName string, Obj *ast.Object) bool {
	rw := RenameWalker{NewName:NewName, Obj:Obj}
	ast.Walk(&rw, file)
	return rw.Updated
}

type RenameWalker struct {
	NewName string
	Updated bool
	Obj *ast.Object
}

func (this *RenameWalker) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.Ident:
		obj, _ := types.ExprType(n, LocalImporter)
		if obj == this.Obj {
			this.Updated = true
			n.Name = this.NewName
		}
	case *ast.SelectorExpr:
		obj, _ := types.ExprType(n, LocalImporter)
		if obj == this.Obj {
			this.Updated = true
			n.Sel.Name = this.NewName
		}
	}
	return this
}
