package pack

import "net"

var S net.IPAddr

type T1 struct {
	A int
	G string
}

type T2Func func(int) T2
type T2 struct {
	A string
	C string
	M int
}

func (t T2) What() {
}
func (t *T2) WhatPointer(_ *int, _ []int) {

}

type T3 struct {
	D string
	M *int
}
type I0 interface {
	M() int
}

func (t T3) What() int {
	return *t.M
}

type I1 interface {
	// X() interface{}
	D(interface{})
	Z()
}
type I2 interface {
	// Y() string
	// I1
	I0
}

type OP struct {
	A int
}

type T4 struct {
	A1  int
	A2  int
	A3  int
	A4  int
	A5  int
	A6  int
	A7  int
	A8  int
	A9  int
	A10 int
	A11 int
	A12 int
}
