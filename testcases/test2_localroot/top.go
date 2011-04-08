package main

import pkg2_0	"pkg2/T"

import (
	"package1"
)

func main() {
	//comment 1
	package1.Foo()
	//comment 2
	var t *pkg2_0.T
	t.Foo()
	t.Bar()
}
