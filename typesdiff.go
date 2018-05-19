package main

import (
	"fmt"
	"go/types"
	"log"
	"strings"
)

// diff do lots of printing
func diff(p1 *types.Package, p2 *types.Package) {

	examinedIdentifiers := make(map[string]bool) //Identifiers we saw in p1
	removed := []string{}                        //identifiers in p1 but not in p2

	for _, p1name := range p1.Scope().Names() {
		obj1 := p1.Scope().Lookup(p1name)
		if !obj1.Exported() { //Check only exported symbols
			continue
		}

		examinedIdentifiers[p1name] = true

		obj2 := p2.Scope().Lookup(p1name)
		if obj2 == nil {
			removed = append(removed, p1name)
			continue

		}

		t1, t2 := obj1.Type().Underlying(), obj2.Type().Underlying() //type.Types for current identifier

		t1StringRepr := strings.TrimPrefix(t1.String(), p1.Name()+".")
		t2StringRepr := strings.TrimPrefix(t2.String(), p2.Name()+".")

		generalPrinting := func() {
			log.Print(prettyPrint(obj2, "+", ""))
			log.Print(prettyPrint(obj1, "-", ""))
		}

		switch obj1.(type) {
		case *types.Var, *types.Const, *types.Func:
			if t1StringRepr != t2StringRepr {
				generalPrinting()
			}

		case *types.TypeName:
			_, t2Type := obj2.(*types.TypeName)
			if !t2Type {
				generalPrinting()
			}

			switch t1.(type) {
			case *types.Struct:
				_, isT2Struct := t2.(*types.Struct)
				// indentedPrint for a struct to struct changes
				if isT2Struct {
					structDiff(t1.(*types.Struct), t2.(*types.Struct), obj1.Name(), obj1.Pkg().Name())
				} else {
					generalPrinting()
				}
				printMethods(obj1.Type().(*types.Named), obj2.Type().(*types.Named), obj1.Name())

			case *types.Interface:
				_, isT2Interface := t2.(*types.Interface)
				// indentedPrint for a interface to interface changes
				if isT2Interface {
					interfaceDiff(t1.(*types.Interface), t2.(*types.Interface), obj1.Name())
				} else {
					generalPrinting()
				}
			default:
				if t1StringRepr != t2StringRepr {
					generalPrinting()
				}
			}

		default:
			log.Println("Not sure of ", obj1, "and", obj2)
		}
	}

	for _, name := range removed {
		log.Print(prettyPrint(p1.Scope().Lookup(name), "-", ""))
	}

	for _, name := range p2.Scope().Names() {
		// added symbols printing
		o2 := p2.Scope().Lookup(name)
		if !o2.Exported() {
			continue
		}
		if _, ok := examinedIdentifiers[name]; ok {
			continue
		}
		log.Print(prettyPrint(p2.Scope().Lookup(name), "+", ""))

	}
}

//Specific printning for structs and interfaces
func printDiff(t1, t2 types.Type, name, pkgName string) {
	switch t1.(type) {
	case *types.Named:
		switch t1.Underlying().(type) {
		case *types.Struct:
			structDiff(t1.Underlying().(*types.Struct), t2.Underlying().(*types.Struct), name, pkgName)

		case *types.Interface:
			interfaceDiff(t1.Underlying().(*types.Interface), t2.Underlying().(*types.Interface), name)

		default:
			log.Println("Unimplemented type", name, t1.Underlying())
		}
	}
}

func structDiff(s1, s2 *types.Struct, name, pkgName string) {
	//TODO: Change from a map to golang.org/x/tools/go/types/typeutil.Map

	var resString strings.Builder
	resString.WriteString(fmt.Sprintf("type %v struct {\n", name))

	s1Fields := map[string]bool{} //Fields s1

	for i := 0; i < s1.NumFields(); i++ {
		if s1.Field(i).Exported() {
			s1Fields[s1.Field(i).String()] = false //The field is in s1 and may be in s2
		}
	}

	for i := 0; i < s2.NumFields(); i++ {
		if s2.Field(i).Exported() {
			if _, ok := s1Fields[s2.Field(i).String()]; !ok { //The field is not present in s1
				//TODO: check if type printing is more redeable
				resString.WriteString(fmt.Sprintf("\t+%v %v\n", s2.Field(i).Name(), s2.Field(i).Type()))
			} else {
				s1Fields[s2.Field(i).String()] = true
			}
		}
	}

	for field, state := range s1Fields {
		if state == false {
			full := fmt.Sprintf("\t-%v", strings.Replace(field, "field ", "", 1))
			full = strings.Replace(full, pkgName+".", "", 1)
			resString.WriteString(full + "\n")
		}
	}

	//Write data only if other things has been written to the builder
	if resString.String() != fmt.Sprintf("type %v struct {\n", name) {
		resString.WriteString("}")
		log.Println(resString.String())
	}
}

func interfaceDiff(i1, i2 *types.Interface, name string) {
	//TODO: Change from a map to golang.org/x/tools/go/types/typeutil.Map

	var resString strings.Builder
	resString.WriteString(fmt.Sprintf("type %v interface {\n", name))
	type presenceCheck struct {
		both bool
		elem types.Object
	}
	i1Elem := map[string]presenceCheck{}

	for i := 0; i < i1.NumExplicitMethods(); i++ {
		if i1.ExplicitMethod(i).Exported() {
			i1Elem[i1.ExplicitMethod(i).Name()] = presenceCheck{false, i1.ExplicitMethod(i)} //False means that it proved to be in s1 and can be in s2

		}
	}

	for i := 0; i < i2.NumExplicitMethods(); i++ {
		if i2.ExplicitMethod(i).Exported() {
			if _, ok := i1Elem[i2.ExplicitMethod(i).Name()]; !ok { //The field is not present in s1
				resString.WriteString(strings.Replace(prettyPrint(i2.ExplicitMethod(i), "+", "\t"), "("+name+").", "", 1))
			} else {
				i1Elem[i2.ExplicitMethod(i).Name()] = presenceCheck{true, i1Elem[i2.ExplicitMethod(i).Name()].elem}
			}
		}

	}

	//Check for imbedded interfaces
	for i := 0; i < i1.NumEmbeddeds(); i++ {
		if i1.Embedded(i).Obj().Exported() {
			i1Elem[i1.Embedded(i).String()] = presenceCheck{false, i1.Embedded(i).Obj()} //False means that it proved to be in s1 and can be in s2

		}
	}

	for i := 0; i < i2.NumEmbeddeds(); i++ {
		if i2.Embedded(i).Obj().Exported() {
			if _, ok := i1Elem[i2.Embedded(i).String()]; !ok { //The field is not present in s1

				resString.WriteString(strings.Replace(prettyPrint(i2.Embedded(i).Obj(), "+", "\t"), "("+name+").", "", 1))
			} else {
				i1Elem[i1.Embedded(i).String()] = presenceCheck{true, i1Elem[i1.Embedded(i).String()].elem}
			}
		}
	}

	for _, c := range i1Elem {
		if !c.both {
			resString.WriteString(strings.Replace(prettyPrint(c.elem, "-", "\t"), "("+name+").", "", 1))

		}
	}

	//Write data only if other things has been written to the builder
	if resString.String() != fmt.Sprintf("type %v interface {\n", name) {
		resString.WriteString("}")
		log.Println(resString.String())
	}

}

func printMethods(func1 *types.Named, func2 *types.Named, interfaceName string) {

	type presenceCheck struct {
		both bool
		elem types.Object
	}
	count := 0
	const LIMIT = 5
	addCount := func() {
		if count == 0 {
			log.Println(interfaceName)
		}
		count++
	}
	func1Meth := map[string]presenceCheck{} //Methods in func1

	for i := 0; i < func1.NumMethods(); i++ {
		if func1.Method(i).Exported() {
			func1Meth[func1.Method(i).String()] = presenceCheck{false, func1.Method(i)} //The method is in func1 and may be in func2
		}
	}

	for i := 0; i < func2.NumMethods(); i++ {
		if func2.Method(i).Exported() {
			if _, ok := func1Meth[func2.Method(i).String()]; !ok { //The method is not present in func1
				if count < LIMIT {
					addCount()
					log.Printf("\t%v", prettyPrint(func2.Method(i), "+", ""))
				}

			} else {
				func1Meth[func2.Method(i).String()] = presenceCheck{true, func2.Method(i)}
			}
		}
	}

	//Check the ones removed
	for _, state := range func1Meth {
		if !state.both && count < LIMIT {
			addCount()
			log.Printf("\t%v", prettyPrint(state.elem, "-", ""))
		}
	}
	if LIMIT == count {
		log.Println("\t", "Other Methods ....")
	}

}

//Common types general printing
func prettyPrint(obj types.Object, state, padding string) string {
	switch obj.(type) {

	case *types.Var, *types.Const:
		res := fmt.Sprintf("%v%v%v\n", padding, state, strings.Replace(obj.String(), obj.Pkg().Name()+".", "", -1))
		return strings.Replace(res, "untyped ", "", -1)

	//Do functions and methods
	case *types.Func:
		objRepresentation := obj.String()
		objRepresentation = strings.Replace(objRepresentation, obj.Pkg().Name()+".", "", -1)
		return fmt.Sprintf("%v%v%v\n", padding, state, objRepresentation)
	//Provides less info on struct or interface
	//ex:
	//type interface A {A() int; B()}
	//type struct B{....}
	case *types.TypeName:

		objRepresentation := obj.String()
		if len(objRepresentation) > 115 {
			start := strings.Index(objRepresentation, "{")
			end := strings.LastIndex(objRepresentation, "}")
			if start < len(objRepresentation) && end != -1 && start != -1 {

				objRepresentation = objRepresentation[0:start+1] + "...." + objRepresentation[end:]
			} else {
				objRepresentation = strings.Replace(objRepresentation, "{}", "{....}", 1)
			}
		}
		objRepresentation = strings.Replace(objRepresentation, obj.Pkg().Name()+".", "", -1)
		objRepresentation = strings.Replace(objRepresentation, "{", " {", 1)
		objRepresentation = strings.Replace(objRepresentation, "\\\"", "", -1)
		return fmt.Sprintf("%v%v%v\n", padding, state, objRepresentation)

	default:
		return fmt.Sprintf("%v%vUntreated%v\n", padding, state, obj.Name())

	}
}
