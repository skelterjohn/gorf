package main

import (
	pkg1	"pkg1"
	pkg2	"pkg2"
)

func main() {
	pkg1.Foo()
	var t *pkg2.T
	t.Foo()
	t.Bar()
}
