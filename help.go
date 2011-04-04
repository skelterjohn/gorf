// Copyright 2011 John Asmuth. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

func Help(cmd string) string {
	switch cmd {
	case "undo":
		return `Usage: gorf [flags] undo
	"undo" will roll-back one refactoring.	
`
	case "clear":
		return `Usage: gorf [flags] clear
	"clear" will remove all files tracking changes.	
`
	case "changes":
		return `Usage: gorf [flags] changes
	"changes" will list each tracked refactoring. 
`
	case "pkg":
		return `Usage: gorf [flags] pkg <path> <old name> <new name>
	"pkg" will change the name of the package in the specified path.
`
	case "rename":
		return `Usage: gorf [flags] rename <path> [<type>.]<old name> <new name>
	"rename" will change the name of a top-level decleration in the package in the specified path.
`
	case "move":
		return `Usage: gorf [flags] move <old path> <new path> [<name>+]
	"move" will move a package, or (if names are specified) a subset of a package.
`
	case "merge":
		return `Usage: gorf [flags] merge <old path> <new path>
	"merge" will merge two packages.
`
	}
	return "Unknown cmd: "+cmd
}