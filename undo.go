// Copyright 2011 John Asmuth. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"os"
	"regexp"
	"strconv"
	"fmt"
	"path/filepath"
	"strings"
)

func UndoCmd(args []string) (err os.Error) {
	if len(args) != 0 {
		return os.NewError("Usage: gorf [flags] undo")
	}
	
	lastChangePath := filepath.Join(LocalRoot, ".change.0.gorfc")
	
	var srcFile *os.File
	srcFile, err = os.Open(lastChangePath)
	if err != nil {
		return
	}
	
	buf := make([]byte, 1024)
	var n int
	n, err = srcFile.Read(buf)
	fmt.Printf("Undoing \"%s\"\n", strings.TrimSpace(string(buf[:n])))
	
	filepath.Walk(LocalRoot, undoscanner(0), nil)
	
	ur := UndoRoller{incr:-1}
	filepath.Walk(LocalRoot, &ur, nil)
	return ur.err
	
	return
}

type undoscanner int

func (this undoscanner) VisitDir(dpath string, f *os.FileInfo) bool {
	return true
}

func (this undoscanner) VisitFile(fpath string, f *os.FileInfo) {
	if !(
		strings.HasSuffix(fpath, ".0.gorf") ||
		 strings.HasSuffix(fpath, ".0.gorfn") ||
		 strings.HasSuffix(fpath, ".0.gorfc")) {
		return
	}

	dir, file := filepath.Split(fpath)
	if dir == "" {
		dir = "."
	}
	dir = filepath.Clean(dir)


	// the realfile was modified by the last command
	if strings.HasSuffix(file, ".0.gorf") {
		realfile := file[1:len(file)-len(".0.gorf")]
		fmt.Printf("Restoring %s\n", filepath.Join(dir, realfile))
		Copy(fpath, filepath.Join(dir, realfile))
		os.Remove(fpath)
		return
	}

	// the realfile was created by the last command
	if strings.HasSuffix(file, ".0.gorfn") {
		realfile := file[1:len(file)-len(".0.gorfn")]
		fmt.Printf("Removing %s\n", filepath.Join(dir, realfile))
		os.Remove(filepath.Join(dir, realfile))
		os.Remove(fpath)
		return
	}
	
	// this just describes the last change
	if strings.HasSuffix(file, ".0.gorfc") {
		os.Remove(fpath)
		return
	}
}

func ChangesCmd(args []string) (err os.Error) {
	var i int
	for i=0; ; i++ {
		changePath := filepath.Join(LocalRoot, fmt.Sprintf(".change.%d.gorfc", i))
	
		srcFile, err := os.Open(changePath)
		if err != nil {
			break
		}
		
		if i == 0 {
			fmt.Printf("Recent refactorings\n")
			fmt.Printf("Age\tCommand\n")		
		}
		
		buf := make([]byte, 1024)
		var n int
		n, err = srcFile.Read(buf)
		change := strings.TrimSpace(string(buf[:n]))
		
		fmt.Printf("%d:\t%s\n", i, change)
	}
	if i == 0 {
		fmt.Printf("No refactorings found\n")
	}
	return
}

func ClearCmd(args []string) (err os.Error) {
	filepath.Walk(LocalRoot, UndoRemover(0), nil)
	return 
}
type UndoRemover int

func (this UndoRemover) VisitDir(dpath string, f *os.FileInfo) bool {
	return true
}

func (this UndoRemover) VisitFile(fpath string, f *os.FileInfo) {
	if !(
		strings.HasSuffix(fpath, ".gorf") ||
		 strings.HasSuffix(fpath, ".gorfn") ||
		 strings.HasSuffix(fpath, ".gorfc")) {
		return
	}
	os.Remove(fpath)
	return
}

func RollbackUndos() (err os.Error) {
	ur := UndoRoller{incr:1}
	filepath.Walk(LocalRoot, &ur, nil)
	return ur.err
}

type UndoRoller struct {
	incr int
	err os.Error
}

func (this *UndoRoller) VisitDir(dpath string, f *os.FileInfo) bool {
	return true
}

var (
	GorfRE = regexp.MustCompile(`\.(.+)\.([0-9]+)\.(gorf)`)
	GorfNRE = regexp.MustCompile(`\.(.+)\.([0-9]+)\.(gorfn)`)
	GorfCRE = regexp.MustCompile(`\.(.+)\.([0-9]+)\.(gorfc)`)
) 

func (this *UndoRoller) VisitFile(fpath string, f *os.FileInfo) {
	dir, name := filepath.Split(fpath)
	var realname, ext string
	var version int
	if toks := GorfRE.FindStringSubmatch(name); toks != nil {
		realname = toks[1]
		version, _ = strconv.Atoi(toks[2])
		ext = toks[3]
	}
	if toks := GorfNRE.FindStringSubmatch(name); toks != nil {
		realname = toks[1]
		version, _ = strconv.Atoi(toks[2])
		ext = toks[3]
	}
	if toks := GorfCRE.FindStringSubmatch(name); toks != nil {
		realname = toks[1]
		version, _ = strconv.Atoi(toks[2])
		ext = toks[3]
	}
	if realname == "" {
		return
	}
	
	version += this.incr
	
	npath := filepath.Join(dir, fmt.Sprintf(".%s.%d.%s", realname, version, ext))
	//fmt.Printf("%s->%s\n", fpath, npath)
	err := os.Rename(fpath, npath)
	if err != nil {
		this.err = err
		fmt.Printf("Error during rollback: %s\n", err)
	}
	
}
