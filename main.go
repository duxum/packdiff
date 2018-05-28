package main

import (
	"flag"
	"fmt"
	"go/ast"
	"go/build"
	"go/importer"
	"go/parser"
	"go/token"
	"go/types"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// To be used thoughtout the program
var fset = token.NewFileSet()

const mode = parser.ParseComments

func main() {
	log.SetFlags(0)
	pkgPath := flag.String("p", "", "Path to the package to analyze from GOPATH")
	flag.Parse()
	if *pkgPath == "" {
		log.Println("A file to diff is required")
		flag.PrintDefaults()
		return

	}
	*pkgPath = strings.Replace(*pkgPath, "/", string(os.PathSeparator), -1)
	tag1, tag2, err := getTags()
	if err != nil {
		log.Println(err)
		return
	}
	sourcePath := filepath.Join(getGOPATH(), "src")

	thePkgPath := filepath.Join(sourcePath, *pkgPath)
	if _, err := os.Stat(thePkgPath); os.IsNotExist(err) {
		log.Println("package do not exist")
		return
	}

	p1Directory, p2Directory, err := getTempDir(thePkgPath, tag1, tag2) //Directories for the two package to analyze
	if err != nil {
		log.Print(err)
		return
	}

	//Delete of created directories
	//If the first tag is empty(current), only one temporary directory is created and is to be deleted
	// Otherwise two are created
	rootDir := filepath.Base(gitRoot(thePkgPath))
	toDelete1 := p1Directory[:strings.Index(p1Directory, rootDir)] + rootDir
	toDelete2 := p2Directory[:strings.Index(p1Directory, rootDir)] + rootDir

	if tag1 == "current" {
		defer os.RemoveAll(filepath.Dir(toDelete2))
	} else {
		defer os.RemoveAll(filepath.Dir(toDelete1))
		defer os.RemoveAll(filepath.Dir(toDelete2))
	}

	p1, err := getPackage(p1Directory)
	if err != nil {
		log.Println(err)
		return
	}
	p2, err := getPackage(p2Directory)

	if err != nil {
		log.Println(err)
		return
	}
	diff(p1, p2)
}

type fileOperationResult struct {
	path string
	err  error
}

//getTempDir returns paths to directories of packages to be diff
//Temp directories are deleted in case of an error
func getTempDir(pkgPath, tag1, tag2 string) (string, string, error) {

	isGit := isDirectoryGit(pkgPath)

	if !isGit {
		return "", "", fmt.Errorf("cannot work in a non git directory %v", pkgPath)
	}

	//Result directories with package copied and appropriate tags checked out
	var d1CopyResult, d2CopyResult fileOperationResult

	numOfDirToMake := 2

	p1Chan := make(chan fileOperationResult, 1)
	p2Chan := make(chan fileOperationResult, 1)

	//Create one directory when one tag is master and use master as base for p1
	if tag1 == "current" {
		numOfDirToMake = 1
		d1CopyResult = fileOperationResult{path: pkgPath, err: nil}
	} else {
		go copyAndCheckout(pkgPath, tag1, p1Chan)
	}
	go copyAndCheckout(pkgPath, tag2, p2Chan)

	for i := 0; i < numOfDirToMake; i++ {

		select {
		case d1CopyResult = <-p1Chan:
		case d2CopyResult = <-p2Chan:
		}
	}
	// Delete of temporary folders in case of an error
	if d1CopyResult.err != nil || d2CopyResult.err != nil {
		os.RemoveAll(d1CopyResult.path)
		os.RemoveAll(d2CopyResult.path)
		err := d1CopyResult.err
		if err == nil {
			err = d2CopyResult.err
		}
		return "", "", err
	}

	return d1CopyResult.path, d2CopyResult.path, nil

}

//copyAndCheckout create a temporary clone directory of git repo with appropriate tag
//in case of error it deletes nothing
func copyAndCheckout(pkgPath string, tag string, out chan fileOperationResult) {
	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		out <- fileOperationResult{path: tmp, err: fmt.Errorf("error creating temporary directory: %v", err)}
		return
	}
	root := gitRoot(pkgPath) //git initialization dir

	cmd := exec.Command("git", "clone", "--local", root)
	cmd.Dir = tmp
	output, err := cmd.CombinedOutput()
	if err != nil {
		out <- fileOperationResult{path: tmp, err: fmt.Errorf("error cloning into temporary folder: %v", string(output))}
		return
	}

	//Checking out to the  required tag
	cmd = exec.Command("git", "checkout", tag)
	cmd.Dir = filepath.Join(tmp, filepath.Base(root))
	output, err = cmd.CombinedOutput()
	if err != nil {
		out <- fileOperationResult{path: tmp, err: fmt.Errorf(string(output))}
		return
	}

	var pathFromRoot string
	if pkgPath != root {
		pathFromRoot = pkgPath[len(root)+1:]
	}
	out <- fileOperationResult{path: filepath.Join(tmp, filepath.Base(root), pathFromRoot), err: nil}

}

//getPackage returns the package in a directory as long as there is a single package in that directory
func getPackage(pkgPath string) (*types.Package, error) {

	//ignore test files, other files and conditional compiled windows files
	bFiles, err := getBuildFiles(pkgPath)
	if err != nil {
		return nil, err
	}

	bFilesMap := make(map[string]bool)

	for _, bFile := range bFiles {
		bFilesMap[bFile] = true
	}
	var filterFunc = func(info os.FileInfo) bool {
		_, ok := bFilesMap[info.Name()]
		return ok
	}

	pkgs, err := parser.ParseDir(fset, pkgPath, filterFunc, mode)
	if err != nil {
		return nil, fmt.Errorf("error parsing the directory: %v, %v", err, pkgPath)
	}

	if len(pkgs) != 1 {
		pkgsSlice := []string{}
		for _, pkgName := range pkgs {
			pkgsSlice = append(pkgsSlice, pkgName.Name)
		}
		return nil, fmt.Errorf("cannot have %v packages in the same directories", pkgsSlice)
	}

	// pkgName := path.Base(pkgPath) //Get pckage name from base name of the directory
	var pkgName string

	for name := range pkgs {
		pkgName = name
	}

	files := []*ast.File{}
	for _, file := range pkgs[pkgName].Files {
		files = append(files, file)
	}

	messagesCount := 0
	const LIMIT = 4

	errorMessage := func(err error) {
		if messagesCount <= LIMIT {
			message := fmt.Sprintf(`%vNOTE: ...%v`, "\t", err.Error()[len(pkgPath):])
			message = strings.Replace(message, "\n", "\n\t", -1)
			fmt.Println(message + "\n")
			messagesCount++
		}
	}

	conf := types.Config{Importer: importer.For("source", nil), Error: errorMessage}
	pkg, err := conf.Check(pkgName, fset, files, nil)
	if err != nil {
		errMsg := strings.Replace(err.Error(), pkgPath, "...", 1)
		log.Printf("Err %v occured during checking of the package at %v", errMsg, pkgPath)
	}

	return pkg, nil
}

func getTags() (tag1 string, tag2 string, err error) {
	if len(os.Args) < 4 {
		err = fmt.Errorf("Check the number of arguments")
		return
	}
	if len(os.Args) == 4 {
		tag1 = "current"
		tag2 = os.Args[3]
		return
	}
	tag1 = os.Args[3]
	tag2 = os.Args[4]
	return
}

//isDirectoryGit returns if a directory is git tracked
func isDirectoryGit(path string) bool {
	cmd := exec.Command("git", "-C", ".", "rev-parse")

	cmd.Dir = path
	output, err := cmd.Output()

	if err != nil || strings.Contains(string(output), "Not a git repository") {
		return false
	}

	return true
}

func getGOPATH() (GOPATH string) {
	if GOPATH = os.Getenv("GOPATH"); GOPATH == "" {
		GOPATH = filepath.Join(os.Getenv("HOME"), "go")
		return
	}
	return
}

//gitRoot returns a directory in which git initialization took place for a git repository it
func gitRoot(root string) string {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = root
	output, _ := cmd.Output()

	outputString := strings.TrimSpace(string(output))
	outputString = strings.Replace(outputString, "/", string(os.PathSeparator), -1)
	return outputString

}

func getBuildFiles(path string) ([]string, error) {
	pkg, err := build.ImportDir(path, build.ImportComment)
	if err != nil {
		return nil, fmt.Errorf("Cannot get build files: %v", err)
	}

	return pkg.GoFiles, nil
}
