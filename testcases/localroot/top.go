package main

import (
	"package1"
	"pkg2"
)

func main() {
	package1.Foo()
	var t *pkg2.T
	t.Foo()
	t.Bar()
}