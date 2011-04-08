package main

import (
	"pkg1"
	"pkg2"
)

func main() {
	//comment 1
	pkg1.Foo()
	//comment 2
	var t *pkg2.T
	t.Foo()
	t.Bar()
}
