package test

import "place2"

var Q place2.MyType

func Foo() {
	place2.Bar()
	if true {
		println(place2.X)
		println(place2.MyType(1))
		println(place2.CX)
		var place struct{ Bar func() }
		place = struct{ Bar func() }{func() {
			println("a")
		}}
		place.Bar()
	}
	println(place2.Z)
	place2.Bar()
}
