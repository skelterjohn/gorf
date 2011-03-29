package test

import "place"

var Q place.MyType

func Foo() {
	place.Baz()
	if true {
		println(place.X)
		println(place.MyType(1))
		println(place.CX)
		var place struct{ Bar func() }
		place = struct{ Bar func() }{func() {
			println("a")
		}}
		place.Bar()
	}
	println(place.Z)
	place.Baz()
}
