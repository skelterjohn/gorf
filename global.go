package main

import (
	//"fmt"
	"os"
	"go/ast"
	"go/token"
	"rog-go.googlecode.com/hg/exp/go/types"
)	

func VarCmd(args []string) (err os.Error) {
	return GlobalCmd("var", args)
}

func ConstCmd(args []string) (err os.Error) {
	return GlobalCmd("const", args)
}

func FuncCmd(args []string) (err os.Error) {
	return GlobalCmd("func", args)
}

func TypeCmd(args []string) (err os.Error) {
	return GlobalCmd("type", args)
}

func GlobalCmd(kind string, args []string) (err os.Error) {
	if len(args) != 3 {
		return MakeErr("Usage: gorf [flags] %s <path> <old name> <new name>", kind)
	}
	path, oldname, newname := args[0], args[1], args[2]
	
	if !IsLegalIdentifier(oldname) {
		return MakeErr("Old name %s is not a legal identifier", oldname)
	}
	if !IsLegalIdentifier(newname) {
		return MakeErr("New name %s is not a legal identifier", newname)
	}
	if oldname == newname {
		return MakeErr("Old name and new name are the same")
	}
	
	ScanAllForImports(LocalRoot)
	
	defer func() {
		if err != nil {
			 UndoCmd([]string{})
		}
	}()
	
	pkg := LocalImporter(path)
	
	if pkg == nil {
		return MakeErr("No package found in %s", path)
	}
	
	updated := false
	
	var Obj *ast.Object
	
	for fpath, file := range pkg.Files {
		switch kind {
		case "var":
			vdl := VarConstDeclFinder{oldname:oldname, newname:newname, TokenType:token.VAR}
			ast.Walk(&vdl, file)
		
			if vdl.NameExists {
				return MakeErr("Name %s already exists", newname)
			}
			Obj = vdl.Obj
			
			if vdl.Updated {
				updated = true	
			}
		case "const":
			vdl := VarConstDeclFinder{oldname:oldname, newname:newname, TokenType:token.CONST}
			ast.Walk(&vdl, file)
			
			if vdl.NameExists {
				return MakeErr("Name %s already exists", newname)
			}
			Obj = vdl.Obj
		
			if vdl.Updated {
				updated = true	
			}
		case "func":
			fdl := FuncDeclFinder{oldname:oldname, newname:newname}
			ast.Walk(&fdl, file)
			
			if fdl.NameExists {
				return MakeErr("Name %s already exists", newname)
			}
			Obj = fdl.Obj
		
			if fdl.Updated {
				updated = true	
			}
		case "type":
			fdl := TypeDeclFinder{oldname:oldname, newname:newname}
			ast.Walk(&fdl, file)
			
			if fdl.NameExists {
				return MakeErr("Name %s already exists", newname)
			}
			Obj = fdl.Obj
		
			if fdl.Updated {
				updated = true	
			}
		}
		
		if updated {
			RenameInFile(file, newname, Obj)
			RewriteSource(fpath, file)
		}
	}
	
	if updated {
		err = RenameInAll(path, newname, Obj)
	}
	
	return
}

type VarConstDeclFinder struct {
	oldname, newname string
	NameExists bool
	Updated bool
	Obj *ast.Object
	Name *ast.Ident
	TokenType token.Token
}

func (this *VarConstDeclFinder) Visit(node ast.Node) ast.Visitor {
	if this.Obj != nil {
		return nil
	}
	switch n := node.(type) {
	case *ast.BlockStmt:
		return nil
	case *ast.GenDecl:
		if n.Tok != this.TokenType {
			return nil
		}
		for _, sp := range n.Specs {
			if vsp, ok := sp.(*ast.ValueSpec); ok {
				for _, name := range vsp.Names {
					if name.Name == this.newname {
						this.NameExists = true
						return nil
					}
					if name.Name == this.oldname {
						this.Name = name
						//name.Name = this.newname
						this.Obj, _ = types.ExprType(name, LocalImporter)
						this.Updated = true
						return nil
					}
				}
			}
		}
		return nil
	}
	return this
}


type FuncDeclFinder struct {
	oldname, newname string
	NameExists bool
	Updated bool
	Obj *ast.Object
	Name *ast.Ident
}

func (this *FuncDeclFinder) Visit(node ast.Node) ast.Visitor {
	if this.Obj != nil {
		return nil
	}
	switch n := node.(type) {
	case *ast.BlockStmt:
		return nil
	case *ast.FuncDecl:
		if n.Name.Name == this.newname {
			this.NameExists = true
			return nil
		}
		if n.Name.Name == this.oldname {
			this.Name = n.Name
			this.Obj, _ = types.ExprType(n.Name, LocalImporter)
			this.Updated = true
			return nil
		}
	}
	return this
}

type TypeDeclFinder struct {
	oldname, newname string
	NameExists bool
	Updated bool
	Obj *ast.Object
	Name *ast.Ident
}

func (this *TypeDeclFinder) Visit(node ast.Node) ast.Visitor {
	if this.Obj != nil {
		return nil
	}
	switch n := node.(type) {
	case *ast.BlockStmt:
		return nil
	case *ast.TypeSpec:
		if n.Name.Name == this.newname {
			this.NameExists = true
			return nil
		}
		if n.Name.Name == this.oldname {
			this.Name = n.Name
			this.Obj, _ = types.ExprType(n.Name, LocalImporter)
			this.Updated = true
			return nil
		}
	}
	return this
}
