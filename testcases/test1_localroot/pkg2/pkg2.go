package pkg2

import (
	fmt	"fmt"
	pkg1	"pkg1"
)

func Bar() {
	fmt.Println("pkg2::Bar")
}

type T struct{ A, b int }

func (t *T) Foo() {
	pkg1.Foo()
}
func (t *T) Bar() {
	Bar()
}
func Baz() {
	fmt.Println("pkg2::Baz")
}
