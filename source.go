package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"go/ast"
	"go/parser"
	"go/token"
	"go/printer"
)

var (
	AllSources = make(map[string]*ast.File)
)

func ParseSource(fpath string) (err os.Error) {
	var ft *ast.File
	ft, err = parser.ParseFile(token.NewFileSet(), fpath, nil, 0)
	if err != nil {
		fmt.Println(err)
		return
	}
	AllSources[fpath] = ft
	return
}



func Copy(cwd, src, dst string) (err os.Error) {
	srcpath := filepath.Join(cwd, src)

	if Verbose {
		fmt.Printf("Copying %s to %s\n", src, dst)
	}

	dstpath := dst
	if !filepath.IsAbs(dstpath) {
		dstpath = filepath.Join(cwd, dst)
	}

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

func RewriteSource(fpath string, ft *ast.File) (err os.Error) {
	dir, name := filepath.Split(fpath)
	backup := ".gorf."+name
	err = Copy(dir, name, backup)
	if err != nil {
		return
	}
	
	var out io.Writer
	out, err = os.Open(fpath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
	if err != nil {
		return
	}
	
	err = printer.Fprint(out, token.NewFileSet(), ft)
	return
}

func GetSourcePackageName(filepath string) (name string, err os.Error) {

	ft := AllSources[filepath]
	if ft == nil {
		err = os.NewError("no such source: "+filepath)
		return
	}
	w := &GetPackageWalker{}

	ast.Walk(w, ft)
	
	name = w.name
	
	return
}

type GetPackageWalker struct {
	name string
}

func (w *GetPackageWalker) Visit(node ast.Node) (v ast.Visitor) {
	switch n := node.(type) {
	case *ast.File:
		w.name = n.Name.Name
		return nil
	}
	return w
}