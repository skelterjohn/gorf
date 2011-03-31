// Copyright 2011 John Asmuth. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"io"
	"path/filepath"
	"os"
	"go/ast"
	"go/printer"
	"go/token"
)

func Copy(srcpath, dstpath string) (err os.Error) {
	var srcFile *os.File
	srcFile, err = os.Open(srcpath, os.O_RDONLY, 0)
	if err != nil {
		return
	}

	var dstFile *os.File
	dstFile, err = os.Open(dstpath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
	if err != nil {
		return
	}

	io.Copy(dstFile, srcFile)

	dstFile.Close()
	srcFile.Close()

	return
}

func BackupSource(fpath string) (err os.Error) {
	dir, name := filepath.Split(fpath)
	backup := "."+name+".gorf"
	err = Copy(fpath, filepath.Join(dir, backup))
	return
}

func Touch(fpath string) {
	f, _ := os.Open(fpath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
	f.Close()
}

func MoveSource(oldpath, newpath string) (err os.Error) {
	if _, e := os.Stat(newpath); e == nil {
		BackupSource(newpath)
	}
	err = BackupSource(oldpath)
	if err != nil {
		return
	}
	dir, file := filepath.Split(newpath)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return
	}
	Touch(filepath.Join(dir, "."+file+".gorfn"))
	err = Copy(oldpath, newpath)
	return
}

func RewriteSource(fpath string, file *ast.File) (err os.Error) {
	err = BackupSource(fpath)
	if err != nil {
		return
	}

	var out io.Writer
	out, err = os.Open(fpath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
	if err != nil {
		return
	}
	
	err = printer.Fprint(out, token.NewFileSet(), file)
	return
}
