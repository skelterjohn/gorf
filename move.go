package main

import (
	"os"
	"go/ast"
	"path/filepath"
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
	
	if PackageTops[newpath] != nil {
		return MakeErr("New path %s already has a package (use merge to combine two packages", newpath)
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
	return
}