package pack

import "io"

func K(int)                             {}
func L(_ string, _ ...int) string       { return "" }
func FuncInterface0(_ io.Reader, _ int) {}
func FuncVariadic0(...int)              {}
func FuncVariadic1(int)                 {}

func I() {}
