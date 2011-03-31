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
	//"gonicetrace.googlecode.com/hg/nicetrace"
	"rog-go.googlecode.com/hg/exp/go/types"
)

var (
	AllSources = make(map[string]*ast.File)
	AllPackages = make(map[string]*ast.Package)
	FileSet = token.NewFileSet()
)

func ParseSource(fpath string) (err os.Error) {
	var ft *ast.File
	ft, err = parser.ParseFile(FileSet, fpath, nil, 0)
	if err != nil {
		return
	}
	AllSources[fpath] = ft
	return
}

func Touch(fpath string) {
	f, _ := os.Open(fpath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
	f.Close()
}

func Copy(srcpath, dstpath string) (err os.Error) {

	if Verbose {
		fmt.Printf("Copying %s to %s\n", srcpath, dstpath)
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

func BackupSource(fpath string) (err os.Error) {
	dir, name := filepath.Split(fpath)
	backup := "."+name+".gorf"
	err = Copy(fpath, filepath.Join(dir, backup))
	return
}

func RewriteSource(fpath string, ft *ast.File) (err os.Error) {
	err = BackupSource(fpath)
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

	name = ft.Name.Name
	
	//ast.Walk(DepthWalker(0), ft)
	
	return
}

type DepthWalker int

func (d DepthWalker) Visit(node ast.Node) ast.Visitor {
	buffer := ""
	for i:=0; i<int(d); i++ {
		buffer += " "
	}
	if node != nil {
		fmt.Printf("%s%T\n", buffer, node)
		fmt.Printf("%s %v\n", buffer, node)
		if e, ok := node.(ast.Expr); ok && e != nil {		
			obj, typ := types.ExprType(e, types.DefaultImporter)
			fmt.Printf("%styp %v\n", buffer, typ)
			if obj != nil {
				fmt.Printf("%sobj: %v\n", buffer, obj)
				fmt.Printf("%sobj.Decl: %T %v\n", buffer, obj.Decl, obj.Decl)
				switch d := obj.Decl.(type) {
				
				}
			}
			fmt.Println()
		}
	}
	
	return d+1
}

/*
	//defer nicetrace.Print()
	if node != nil {
		if e, ok := node.(ast.Expr); ok && e != nil {
			fmt.Printf("%T\n%v\n", node, node)
			obj, typ := types.ExprType(e, types.DefaultImporter)
			fmt.Printf("typ %v\n", typ)
			if obj != nil {
				fmt.Printf("obj: %T %v\n", obj, obj)
				fmt.Printf("obj.Decl: %T %v\n", obj.Decl, obj.Decl)
				switch d := obj.Decl.(type) {
				
				}
			}
			fmt.Println()
		}
	}
*/