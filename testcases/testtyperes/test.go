package test

import "place"

var Q place.MyType2

func Foo() {
	place.Bar()
	if true {
		println(place.X)
		println(place.MyType2(1))
		println(place.CX)
		var place struct{ Bar func() }
		place = struct{ Bar func() }{func() {
			println("a")
		}}
		place.Bar()
	}
	println(place.Z)
	place.Bar()
}
