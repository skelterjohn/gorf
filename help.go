package main

func Help(cmd string) string {
	switch cmd {
	case "undo":
		return `Usage: gorf [flags] undo
	"undo" will roll-back the last refactoring that occured.	
`
	case "scan":
		return `Usage: gorf [flags] scan <path>
	"scan" will print out the ast for the specified path.
`
	case "pkg":
		return `Usage: gorf [flags] pkg <path> <old name> <new name>
	"pkg" will change the name of the package in the specified path.
`
	case "var":
		return `Usage: gorf [flags] var <path> <old name> <new name>
	"var" will change the name of a top-level var in the package in the specified path.
`
	case "const":
		return `Usage: gorf [flags] const <path> <old name> <new name>
	"const" will change the name of a top-level const in the package in the specified path.
`
	case "func":
		return `Usage: gorf [flags] func <path> <old name> <new name>
	"func" will change the name of a functino in the package in the specified path.
`
	case "type":
		return `Usage: gorf [flags] type <path> <old name> <new name>
	"func" will change the name of a type in the package in the specified path.
`
	case "field":
		return `Usage: gorf [flags] field <path> <struct name> <old field name> <new field name>
	"field" will change the name of a struct's field in the package in the specified path.
`
	}
	return "Unknown cmd: "+cmd
}