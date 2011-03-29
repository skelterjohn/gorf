package main

import (
	"fmt"
	"os"
)

var (
	Verbose bool
)

func main() {
	if len(os.Args) < 2 {
		Usage()
		return
	}
	
	var err os.Error
	
	switch os.Args[1] {
	case "target":
		if len(os.Args) != 4 {
			Usage()
			return
		}
		old, new := os.Args[2], os.Args[3]
		err = MvTarget(old, new)
	case "package":
		if len(os.Args) != 5 {
			Usage()
			return
		}
		target, old, new := os.Args[2], os.Args[3], os.Args[4]
		err = ChangePackages(target, old, new)
	case "undo":
		if len(os.Args) != 2 {
			Usage()
			return
		}
		err = Undo()
	}
	
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
	}
}