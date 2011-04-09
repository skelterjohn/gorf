package pkg2

import pkg2_0	"pkg2"
import "package1"

type T struct {
	A, b int
	//comment here too
}

//bring?
func (t *T) Foo() {
	//c1
	package1.Foo()
	//c2
}

func (t *T) Bar() {
	pkg2_0.Bar()
}
