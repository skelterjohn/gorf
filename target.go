package main

import (
	"strings"
	"fmt"
	"os"
	"path/filepath"
)

type Target struct {
	Name, Path string
	Source []string
}

var (
	AllTargets = make(map[string]*Target)
	DirNameTargets = make(map[string]map[string]*Target)
)

func GetDirTargets(dir string) (dts map[string]*Target) {
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
	i, ok := AllTargets[importKey]
	if !ok {
		i = new(Target)
		i.Name, i.Path = name, dir
		AllTargets[importKey] = i
	}
	i.Source = append(i.Source, file)
	
	GetDirTargets(dir)[name] = i
}