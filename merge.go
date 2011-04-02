package main

import (
	"os"
	"fmt"
	"go/ast"
	"path/filepath"
)

func MergeCmd(args []string) (err os.Error) {
	if len(args) != 2 {
		return MakeErr("Usage: gorf [flags] merge <old path> <new path>")
	}

	oldpath, newpath := args[0], args[1]
	
	defer func() {
		if err != nil {
			 UndoCmd([]string{})
		}
	}()
	
	err = ScanAllForImports(LocalRoot)
	if err != nil {
		return
	}
	
	if PackageTops[oldpath] == nil {
		return MakeErr("Old path '%s' contains no package", oldpath)
	}
	
	if PackageTops[newpath] == nil {
		return MakeErr("Old path '%s' contains no package", newpath)
	}
	
	oldpkg, newpkg := LocalImporter(oldpath), LocalImporter(newpath)

	// check for conflicts
	duplicates := false
	for name, oldobj := range oldpkg.Scope.Objects {
		if oldobj.Decl == nil {
			continue
		}
		if newobj, ok := newpkg.Scope.Objects[name]; ok && newobj.Decl != nil {
		
		
			duplicates = true
			
			fmt.Printf("Duplicate name %s\n", name)
			if oldNode, oldOk := oldobj.Decl.(ast.Node); oldOk {
				fmt.Printf(" %s\n", AllSources.Position(oldNode.Pos()))
			} else {
				fmt.Printf("%T\n", oldobj.Decl)
			}
			if newNode, newOk := newobj.Decl.(ast.Node); newOk {
				fmt.Printf(" %s\n", AllSources.Position(newNode.Pos()))
			} else {
				fmt.Printf("%T\n", newobj.Decl)
			}
		}
	}
	if duplicates {
		return MakeErr("Packages in '%s' and '%s' contain duplicate names", oldpath, newpath)
	}
	
	//move source files
	for fpath := range oldpkg.Files {
		_, fname := filepath.Split(fpath)
		npath := GetUniqueFilename(newpkg, filepath.Join(newpath, fname))
		
		err = MoveSource(fpath, npath)
		if err != nil {
			return
		}
	}

	//update imports
	for _, ipath := range ImportedBy[QuotePath(oldpath)] {
		pkg := LocalImporter(ipath)
		for fpath, file := range pkg.Files {
			ir := ImportRepath {
				OldName : oldpkg.Name,
				OldPath : oldpath,
				NewPath : newpath,
			}
			ast.Walk(&ir, file)
			if ir.Updated {
				err = RewriteSource(fpath, file)
				if err != nil {
					return
				}
			}
		}
	}

	return// MakeErr("not implemented yet")
}

type ImportRepath struct {
	OldName string
	OldPath, NewPath string
	Updated bool
}

func (this *ImportRepath) Visit(node ast.Node) ast.Visitor {
	if is, ok := node.(*ast.ImportSpec); ok {
		if is.Path.Value == QuotePath(this.OldPath) {
			is.Path.Value = QuotePath(this.NewPath)
			if is.Name == nil {
				is.Name = &ast.Ident{ Name : this.OldName }
			}
			this.Updated = true
			return nil
		}
	}
	return this
}
