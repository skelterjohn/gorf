// Copyright 2011 John Asmuth. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"os"
	"go/ast"
	"path/filepath"
	"rog-go.googlecode.com/hg/exp/go/types"
)

func PkgCmd(args []string) (err os.Error) {
	if len(args) != 2 {
		return MakeErr("Usage: gorf [flags] pkg <path> <new name>")
	}
	
	path, newname := filepath.Clean(args[0]), args[1]
	
	if !IsLegalIdentifier(newname) {
		return MakeErr("New name %s is not a legal identifier", newname)
	}
	
	err = ScanAllForImports(LocalRoot)
	if err != nil {
		return
	}
	
	PreloadImportedBy(path)
	
	defer func() {
		if err != nil {
			 UndoCmd([]string{})
		}
	}()
	
	if PackageTops[path] == nil {
		return MakeErr("No local package found in %s", path)
	}
	
	pkg := LocalImporter(path)
	
	oldname := pkg.Name
	
	for fpath, file := range pkg.Files {
		file.Name.Name = newname
		err = RewriteSource(fpath, file)
		if err != nil {
			return
		}
	}
	
	
	
	for _, ip := range ImportedBy[QuotePath(path)] {
		ipkg := LocalImporter(ip)
		if ipkg == nil {
			return MakeErr("Problem getting package in %s", ip) 
		}
		for fpath, file := range ipkg.Files {
			uniqueName := GetUniqueIdent([]*ast.File{file}, newname)
			
			if uniqueName != newname {
				fmt.Printf("In %s: possible conflict with %s, using %s instead\n", fpath, newname, uniqueName)
			}
			
			pc := PkgChanger {
				path:path,
				oldname:oldname,
				newname:uniqueName,
				pkgname:newname,
			}
			ast.Walk(&pc, file)
			if pc.Updated {
				RewriteSource(fpath, file)
			}
		}
	}
	
	return
}

type PkgChanger struct {
	path string
	oldname, newname string
	pkgname string
	Renamed bool
	Updated bool
}

func (this *PkgChanger) Visit(node ast.Node) ast.Visitor {
	if this.Renamed {
		return nil
	}
	
	if node == nil {
		return this
	}
	
	switch n := node.(type) {
	case *ast.ImportSpec:
		if string(n.Path.Value) == QuotePath(this.path) {
			if n.Name != nil {
				
				this.Renamed = true
				return nil
			} 
			
			if n.Name == nil && this.newname != this.pkgname {
				n.Name = &ast.Ident {
					Name : this.newname,
					NamePos : n.Pos(),
				}
				this.Updated = true
			}
		}
		
		
	case *ast.Ident:
		if n.Name == this.oldname {
			obj, typ := types.ExprType(n, LocalImporter)
			if obj == nil {
				n.Name = this.newname
				this.Updated = true
			}
			if obj != nil {
				if typ.Kind == ast.Pkg && obj.Name == this.oldname {
					n.Name = this.newname
					this.Updated = true	
				}
			}
		}
		
		
	}
	
	return this	
}
