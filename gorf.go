package main

import (
	"os"
	"fmt"
	"flag"
	"gonicetrace.googlecode.com/hg/nicetrace"
)

var (
	LocalRoot = "."
	Usage bool
)

var UsageText = `Usage: gorf [flags] <command>
commands:
  scan <path>
  pkg <path> <old name> <new name>
  var <path> <old name> <new name>
  const <path> <old name> <new name>
  type <path> <old name> <new name>
  func <path> <old name> <new name>
  field <path> <type name> <old field name> <new field name>
  undo
`

func MakeErr(format string, args ...interface{}) (os.Error) {
	return os.NewError(fmt.Sprintf(format, args...))
}

func main() {
	defer nicetrace.Print()
	
	flag.StringVar(&LocalRoot, "r", ".", "Local package root")
	flag.BoolVar(&Usage, "?", false, "Print usage and quit")
	
	flag.Parse()
	
	cmds := map[string]func([]string) os.Error {
		"undo" : UndoCmd,
		"pkg" : PkgCmd,
		"var" : VarCmd,
		"const" : ConstCmd,
		"func" : FuncCmd,
		"type" : TypeCmd,
		"field" : FieldCmd,
		"scan" : ScanCmd,
	}
	
	foo, ok := cmds[flag.Arg(0)]
	if ok && Usage {
		fmt.Println(Help(flag.Arg(0)))
		return
	}

	erf := func(format string, args ...interface{}) {
		fmt.Fprintf(os.Stderr, format, args...)
	}
			
	if !ok || Usage || len(flag.Args()) == 0 {
		erf(UsageText)
		erf("flags\n")
		flag.PrintDefaults()
		return
	}
	
	err := foo(flag.Args()[1:])
	if err != nil {
		erf("%v\n", err)
	}
}