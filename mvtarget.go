package main

import (
	"fmt"
	"os"
	"path/filepath"
	"go/ast"
)

func Exists(fp string) bool {
	_, err := os.Stat(fp)
	return err == nil
}

func MvTarget(oldpath, newpath string) (err os.Error) {
	ScanForTargets()
	
	dirTargets := GetDirTargets(oldpath)
	if len(dirTargets) == 0 {
		err = os.NewError(fmt.Sprintf("No target found in '%s'", oldpath))
		return
	}
	
	blocking := GetDirTargets(newpath)
	
	for name, dirTarget := range dirTargets {
		if _, ok := blocking[name]; ok {
			err = os.NewError(fmt.Sprintf("There is already package %s in '%s'", name, newpath))
			return
		}
		for _, src := range dirTarget.Source {
			_, sf := filepath.Split(src)
			nsf := filepath.Join(newpath, sf)
			if Exists(nsf) {
				err = os.NewError(fmt.Sprintf("There is already file %s in '%s'", sf, newpath))
				return
			}
		}
	}
	
	for _, dirTarget := range dirTargets {
		for _, src := range dirTarget.Source {
			BackupSource(filepath.Join(dirTarget.Path, src))
		}
	}
	
	err = ChangeImportPaths(oldpath, newpath)
	if err != nil {
		return
	}
	
	os.MkdirAll(newpath, 0755)
	
	
	for _, dirTarget := range dirTargets {
		for _, src := range dirTarget.Source {
			sf := filepath.Join(dirTarget.Path, src)
			nsf := filepath.Join(newpath, src)
			Copy(sf, nsf)
			os.Remove(sf)
			Touch(filepath.Join(newpath, "."+src+".gorfn"))
		}
	}
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
		if string(n.Path.Value) == QuoteTarget(w.old) {
			n.Path.Value = []byte(QuoteTarget(w.new))
			w.changed = true
			return nil
		}
	}
	return w
}