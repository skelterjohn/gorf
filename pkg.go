package main

import (
	//"fmt"
	"os"
	"go/ast"
	"rog-go.googlecode.com/hg/exp/go/types"
)

func PkgCmd(args []string) (err os.Error) {
	if len(args) != 3 {
		return MakeErr("Usage: gorf [flags] pkg <path> <old name> <new name>")
	}
	
	path, oldname, newname := args[0], args[1], args[2]
	
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
	
	if pkg.Name != oldname {
		return MakeErr("Package name and old name don't match (%s != %s)", pkg.Name, oldname)
	}
	
	
	for fpath, file := range pkg.Files {
		file.Name.Name = newname
		err = RewriteSource(fpath, file)
		if err != nil {
			return
		}
	}
	
	
	
	for _, ip := range ImportedBy[QuotePath(path)] {
		ipkg := LocalImporter(ip)
		for fpath, file := range ipkg.Files {
			pc := PkgChanger{
				path:path,
				oldname:oldname,
				newname:newname,
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
				if n.Name.Name == this.oldname {
					n.Name.Name = this.newname
					this.Updated = true
				} else {
					this.Renamed = true
					return nil
				}
			}
		}
	case *ast.Ident:	
		if n.Name == this.oldname {
			_, typ := types.ExprType(n, LocalImporter)
			if typ.Kind == ast.Pkg {
				n.Name = this.newname
				this.Updated = true
			}
		}
	}
	
	return this	
}
