package main

import (
	"os"
	//"fmt"
	"go/token"
	//"go/parser"
	"go/ast"
	"path/filepath"
	"log"
	"strings"
	"rog-go.googlecode.com/hg/exp/go/types"
	"rog-go.googlecode.com/hg/exp/go/parser"
)

var (
	AllSourceTops = token.NewFileSet()
	AllSources = token.NewFileSet()
	ImportedBy = make(map[string][]string)
	PackageTops = make(map[string]*ast.Package)
	Packages = make(map[string]*ast.Package)
)

func LocalImporter(path string) (pkg *ast.Package) {
	//fmt.Printf("Importing %s\n", path)
	var ok bool
	var pkgtop *ast.Package
	if pkgtop, ok = PackageTops[path]; !ok {
		pkg = types.DefaultImporter(path)
		return
	}
	if pkg, ok = Packages[path]; ok {
		return
	}
	var sourcefiles []string
	for srcfile := range pkgtop.Files {
		sourcefiles = append(sourcefiles, srcfile)
	}
	//fmt.Printf("Parsing %v\n", sourcefiles)
	dirpkgs, err := parser.ParseFiles(AllSources, sourcefiles, parser.DeclarationErrors)
	if err != nil {
		log.Println(err)
		return
	}
	
	pkg = dirpkgs[pkgtop.Name]
	
	Packages[path] = pkg
	
	//fmt.Printf("nil: %v name: %s\n", pkg == nil, pkgtop.Name)
	
	return
}

func ScanAllForImports(dir string) {
	filepath.Walk(dir, ScanWalker(0), nil)
}

type ScanWalker int

func (s ScanWalker) VisitDir(path string, f *os.FileInfo) bool {
	ScanForImports(path)
	return true
}

func (s ScanWalker) VisitFile(fpath string, f *os.FileInfo) {
	if strings.HasSuffix(fpath, ".gorg") || strings.HasSuffix(fpath, ".ngorg") {
		os.Remove(fpath)
	}
}

//Look at the imports, and build up ImportedBy
func ScanForImports(path string) {
	sourcefiles := filepath.Glob(filepath.Join(path, "*.go"))
	dirpkgs, err := parser.ParseFiles(AllSourceTops, sourcefiles, parser.ImportsOnly)
	
	if err != nil {
		log.Println(err)
	}
	
	//take the first non-main. otherwise, main is ok.
	var prime *ast.Package
	for name, pkg := range dirpkgs {
		prime = pkg
		if name != "main" {
			break
		}
	}
	
	if prime == nil {
		return
	}
	
	PackageTops[path] = prime
		
	is := make(ImportScanner)
	
	ast.Walk(is, prime)
	
	for imp := range is {
		ImportedBy[imp] = append(ImportedBy[imp], path)
	}
	
	return
}

type ImportScanner map[string]bool

func (is ImportScanner) Visit(node ast.Node) ast.Visitor {
	switch n := node.(type) {
	case *ast.ImportSpec:
		is[string(n.Path.Value)] = true
	}
	return is
}