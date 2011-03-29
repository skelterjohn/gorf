package place

import "fmt"

type MyType int

var ZZ = 3
var (
	XX	int
	W	MyType
)

const CX = 3
const (
	CY = 5
)

func Bar() {
	println("hi")
	X := 3
	_ = X
	Bar := 3
	_ = Bar
	w := MyType(0)
	_ = w
}
func F() {
	XX = 2
	Bar()
	y := Bar
	y()
	z := fmt.Println
	z()
}
