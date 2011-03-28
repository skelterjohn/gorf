package main

import (
	"fmt"
)

var UsageText = `Usage: gorf <command>

commands:
 target <old target> <new target>
 package <target> <old name> <new name>
 global <target> <old name> <new name>
 field <target> <type> <old name> <new name>
 undo
`

func Usage() {
	fmt.Print(UsageText)
}