package main

import (
	"os"
	"path/filepath"
	"strings"
)

func Undo() (err os.Error) {
	errch := make(chan os.Error)
	filepath.Walk(".", undoscanner(0), errch)
	return
}

type undoscanner int

func (this undoscanner) VisitDir(dpath string, f *os.FileInfo) bool {
	return true
}

func (this undoscanner) VisitFile(fpath string, f *os.FileInfo) {
	if !strings.HasSuffix(fpath, ".go") {
		return
	}
	
	dir, file := filepath.Split(fpath)
	if dir == "" {
		dir = "."
	}
	dir = filepath.Clean(dir)
	
	if strings.HasPrefix(file, ".gorf.") {
		realfile := file[len(".gorf."):]
		Copy(dir, file, realfile)
		os.Remove(fpath)
		return
	}
}