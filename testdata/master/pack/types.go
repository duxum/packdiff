package pack

import (
	"io"
	"time"
)

var Me io.Writer
var S time.Time

type T2 struct {
	A interface{}
	C string
	M int
	F I4
}
type T2Func func() T2

func (_ T2) Name() {

}
func (_ T2) What() {

}

type T3 struct {
	D string
	M *int
	// ig io.Writer
	// T1
}

type I1 interface {
	X() interface{}
}
type I2 interface {
	Y() string
	I1
}
type I4 interface {
	A1() int
	A3() int
	A2() int
	A4() int
	A5() int
	A6() int
	A7() int
	A8() int
	A9() int
	A10() int
	A11() int
	A12() int
}
