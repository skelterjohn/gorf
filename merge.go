package main

import (
	"os"
)

func MergeCmd(args []string) (err os.Error) {
	if len(args) != 2 {
		return MakeErr("Usage: gorf [flags] merge <old path> <new path>")
	}

	return MakeErr("not implemented yet")
}
