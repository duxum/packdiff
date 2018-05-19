package main

import (
	"bytes"
	"fmt"
	"log"
	"strings"
	"testing"
)

func TestIsGitDirectory(t *testing.T) {
	cases := []struct {
		path  string
		isGit bool
	}{
		{".", true},
		{"./testdata", true},
		{"~", false},
	}
	for _, c := range cases {
		if got := isDirectoryGit(c.path); got != c.isGit {
			fmt.Println("GOT", got)
			t.Fatalf("isGitDirectory %v: expected %v, got %v", c.path, c.isGit, got)
		}
	}

}

func TestOutput(t *testing.T) {
	present := []string{
		//types
		"-type I4 interface {....}",
		"+type T4 struct {....}",
		"+type T1 struct {A int; G string}",
		"-var Me io.Writer",
		"-var S time.Time",
		"+var S net.IPAddr",
		"type T2 struct {",
		"\t-A interface{}",
		"\t+A string",
		"\t-F I4",
		"+type I0 interface {M() int}",
		"type I1 interface {",
		"\t-func X() interface{}",
		"\t+func D(interface{})",
		"\t+func Z()",
		"type I2 interface {",
		"\t-func Y() string",
		"\t-type I1 interface {X() interface{}}",
		"\t+type I0 interface {M() int}",
		"-type T2Func func() T2",
		"-type T2Func func() T2",
		"+type T2Func func(int) T2",

		//Methods
		"+func (T3).What() int",
		"+func (*T2).WhatPointer(_ *int, _ []int)",
		"+func (T3).What() int",
		"-func (T2).Name()",

		//funcs
		"-func K(int)",
		"+func K(int) string",
		"-func FuncVariadic0(...int)",
		"+func FuncInterface0(_ int)",
		"-func FuncInterface0(_ io.Reader, _ int)",
		"+func FuncInterface0(_ int)",
		//basic
		"-type S2 []string",
		"-var Q int",
		"+var Q float64",
		"-var W string",
		"-var J string",
		"+var J interface{}",
		"-var A int",
		"+var A time.Time",
		"-var U1 []int",
		"+var U1 []string",
		"+var R interface{}",
		"-var OP int",
		"+type OP struct {A int}",
		//
	}
	notPresent := []string{
		//types
		"type T3 struct {",

		//funcs
		"FuncVariadic1",
		"func (T2).What()",
		" I ",
		//basic
		" P ",
		" U ",
		"ignore",
	}
	p1, _ := getPackage("testdata/master/pack")
	p2, _ := getPackage("testdata/v0.0.1/pack")
	buf := bytes.Buffer{}

	log.SetOutput(&buf)

	diff(p1, p2)
	output := buf.String()
	for _, c := range present {
		if !strings.Contains(output, c) {
			t.Errorf("expected to contain %v", c)
		}
	}

	for _, c := range notPresent {
		if strings.Contains(output, c) {
			t.Errorf("expected not to contain %v", c)
		}
	}
}
