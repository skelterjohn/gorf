package main

import (
	"os"
	"path/filepath"
	"strings"
)

func UndoCmd(args []string) (err os.Error) {
	if len(args) != 0 {
		return os.NewError("Usage: gorf [flags] undo")
	}
	filepath.Walk(".", undoscanner(0), nil)
	return
}

type undoscanner int

func (this undoscanner) VisitDir(dpath string, f *os.FileInfo) bool {
	return true
}

func (this undoscanner) VisitFile(fpath string, f *os.FileInfo) {
	if !(strings.HasSuffix(fpath, ".go") ||
			strings.HasSuffix(fpath, ".gorf") ||
			strings.HasSuffix(fpath, ".gorfn")) {
		return
	}

	dir, file := filepath.Split(fpath)
	if dir == "" {
		dir = "."
	}
	dir = filepath.Clean(dir)

	// the realfile was modified by the last command
	if strings.HasSuffix(file, ".gorf") {
		realfile := file[1:len(file)-len(".gorf")]
		Copy(fpath, filepath.Join(dir, realfile))
		os.Remove(fpath)
		return
	}

	// the realfile was created by the last command
	if strings.HasSuffix(file, ".gorfn") {
		realfile := file[1:len(file)-len(".gorfn")]
		os.Remove(filepath.Join(dir, realfile))
		os.Remove(fpath)
		return
	}
}