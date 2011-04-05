package main

import pkg2_0	"pkg2/T"
import (
	"package1"
)

func main() {
	package1.Foo()
	var t *pkg2_0.T
	t.Foo()
	t.Bar()
}
