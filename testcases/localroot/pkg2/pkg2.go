package pkg2

import (
	"fmt"
	"package1"
)

func Bar() {
	fmt.Println("pkg2::Bar")
}

type T struct {
	A, b int
}

func (t *T) Foo() {
	package1.Foo()
}

func (t *T) Bar() {
	Bar()
}
