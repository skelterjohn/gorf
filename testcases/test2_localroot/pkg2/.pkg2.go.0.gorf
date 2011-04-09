package pkg2

import (
	"fmt"
	"package1"
)
//a comment
func Bar() {
	fmt.Println("pkg2::Bar")
}

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
	Bar()
}

func Baz() {
     fmt.Println("pkg2::Baz")
}