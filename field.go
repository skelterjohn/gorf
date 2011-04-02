// Copyright 2011 John Asmuth. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"go/ast"
	"rog-go.googlecode.com/hg/exp/go/types"
)

func FieldCmd(args []string) (err os.Error) {
	if len(args) != 4 {
		return MakeErr("Usage: gorf [flags] field <path> <type name> <old field name> <new field name>")
	}
	path, typename, oldname, newname := args[0], args[1], args[2], args[3]
	
	if !IsLegalIdentifier(typename) {
		return MakeErr("Type name %s is not a legal identifier", oldname)
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
	
	err = ScanAllForImports(LocalRoot)
	if err != nil {
		return
	}
	
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
		vdl := FieldDeclFinder{typename:typename, oldname:oldname, newname:newname}
		ast.Walk(&vdl, file)
	
		if vdl.NameExists {
			return MakeErr("Name %s already exists", newname)
		}
		Obj = vdl.Obj
		
		if vdl.Updated {
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

type FieldDeclFinder struct {
	typename, oldname, newname string
	NameExists bool
	Updated bool
	Obj *ast.Object
	Name *ast.Ident
}

func (this *FieldDeclFinder) Visit(node ast.Node) ast.Visitor {
	if this.Obj != nil {
		return nil
	}
	switch n := node.(type) {
	case *ast.BlockStmt:
		return nil
	case *ast.TypeSpec:
		if n.Name.Name == this.typename {
			if st, ok := n.Type.(*ast.StructType); ok {
				for _, field := range st.Fields.List {
					for _, nid := range field.Names {
						if nid.Name == this.newname {
							this.NameExists = true
							return nil
						}
						if nid.Name == this.oldname {
							nid.Name = this.newname
							this.Obj, _ = types.ExprType(nid, LocalImporter)
							this.Updated = true
							this.Name = nid
							return nil
						}
					}
				}
			}
			return this
		}
		return nil
	}
	return this
}