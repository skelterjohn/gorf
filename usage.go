package main

import (
	"fmt"
)

var UsageText = `Usage: gorf <command>

commands:
 target <old target> <new target>
 package <target> <old name> <new name>
 func <target> <package> <old name> <new name>
 var <target> <package> <old name> <new name>
 const <target> <package> <old name> <new name>
 type <target> <package> <old name> <new name>
 //field <target> <package> <type> <old name> <new name>
 undo
`

func Usage() {
	fmt.Print(UsageText)
}