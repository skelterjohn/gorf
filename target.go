package main

import (
	"strings"
	"fmt"
	"os"
	"path/filepath"
	"go/ast"
	"rog-go.googlecode.com/hg/exp/go/types"
)

type Target struct {
	Name, Path string
	Source []string
	pkg *ast.Package
}

var (
	AllTargets = make(map[string]*Target)
	DirNameTargets = make(map[string]map[string]*Target)
)

func GetDirTargets(dir string) (dts map[string]*Target) {
	dir = filepath.Clean(dir)
	var ok bool
	dts, ok = DirNameTargets[dir]
	if !ok {
		dts = make(map[string]*Target)
		DirNameTargets[dir] = dts
	}
	return
}

func ListTargetsSource() {
	for _, i := range AllTargets {
		fmt.Printf("In \"%s\" package %s\n %v\n", i.Path, i.Name, i.Source)
	}
}

func ScanForTargets() {
	errch := make(chan os.Error)
	filepath.Walk(".", scanner(0), errch)
	//ListTargetsSource()
}

type scanner int

func (this scanner) VisitDir(dpath string, f *os.FileInfo) bool {
	return true
}

func (this scanner) VisitFile(fpath string, f *os.FileInfo) {
	if !strings.HasSuffix(fpath, ".go") {
		return
	}
	
	dir, file := filepath.Split(fpath)
	if dir == "" {
		dir = "."
	}
	dir = filepath.Clean(dir)
	
	if strings.HasPrefix(file, ".gorf.") {
		os.Remove(file)
		return
	}
	if strings.HasPrefix(file, ".gorfn.") {
		os.Remove(file)
		return
	}
	
	err := ParseSource(fpath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}
	
	var name string
	name, err = GetSourcePackageName(fpath)
	if err != nil {
		panic(err)
	}
	
	importKey := name+":"+dir
	t, ok := AllTargets[importKey]
	if !ok {
		t = new(Target)
		t.Name, t.Path = name, dir
		t.pkg = &ast.Package{Name:name, Scope:nil, Files:make(map[string]*ast.File)}
		AllTargets[importKey] = t
	}
	t.Source = append(t.Source, file)
	t.pkg.Files[fpath] = AllSources[fpath]
	
	GetDirTargets(dir)[name] = t
}

func LocalImporter(path string) *ast.Package {
	dirTargets := GetDirTargets(path)
	if dirTargets == nil {
		return types.DefaultImporter(path)
	}
	for k, t := range dirTargets {
		if k != "main" {
			return t.pkg
		}
	}
	return nil
}
