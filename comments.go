// Copyright 2011 John Asmuth. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	//"fmt"
	"go/ast"
)

type CommentTie struct {
	C *ast.Comment
	Before, After ast.Node
}

func GetCommentTies(file *ast.File) (ties []CommentTie) {
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			ties = append(ties, CommentTie{C:c})
		}
	}
	
	if len(ties) == 0 {
		return
	}
	tw := TieWalker{ties:ties, next:0}
	
	ast.Walk(&tw, file)
	/*
	for _, ct := range tw.ties {
		fmt.Printf("%v\n%v\n%v\n\n", ct.Before, ct.C, ct.After)
	}
	*/
	
	return
}

type TieWalker struct {
	ties []CommentTie
	next int
}

func (this *TieWalker) Visit(node ast.Node) ast.Visitor {
	if node == nil {
		return this
	}
	check := func() bool {
		if this.next >= len(this.ties) {
			return false
		}
		ch := this.ties[this.next]
		//fmt.Printf("Checking (%d, %d) vs %d\n", node.Pos(), node.End(), ch.C.Slash)
		if node.End() < ch.C.Slash {
			this.ties[this.next].Before = node
		} else if node.Pos() > ch.C.Slash {
			this.ties[this.next].After = node
			this.next++
			return true
		}
		return false
	}
	
	for check() {}
	
	return this
}
