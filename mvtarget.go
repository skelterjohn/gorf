package main

import (
	"os"
	"go/ast"
)

func MvTarget(oldpath, newpath string) (err os.Error) {
	ScanForTargets()
	err = ChangeImportPaths(oldpath, newpath)
	return
}

func ChangeImportPaths(oldpath, newpath string) (err os.Error) {
	for fp, ft := range AllSources {
		var changed bool
		changed, err = ChangeImportPath(oldpath, newpath, ft)
		if changed {
			err = RewriteSource(fp, ft)
		}
	}
	return
}

func ChangeImportPath(oldpath, newpath string, ft *ast.File) (changed bool, err os.Error) {
	oldpath = "\""+oldpath+"\""
	newpath = "\""+newpath+"\""
	ipc := &ImportPathChanger{old:oldpath, new:newpath}
	ast.Walk(ipc, ft)
	changed = ipc.changed
	return
}

type ImportPathChanger struct {
	old, new string
	changed bool
}

func (w *ImportPathChanger) Visit(node ast.Node) (v ast.Visitor) {
	switch n := node.(type) {
	case *ast.ImportSpec:
		if string(n.Path.Value) == w.old {
			n.Path.Value = []byte(w.new)
			w.changed = true
			return nil
		}
	}
	return w
}