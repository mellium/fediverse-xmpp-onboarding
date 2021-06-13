// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:build dev
// +build dev

package main

import (
	"html/template"
	"os"
)

var (
	tmpls  = os.DirFS("./pages/")
	assets = os.DirFS("./static/")
)

var pagesGlob = "*"

func reparseTemplates(*template.Template) (*template.Template, error) {
	return parseTemplates()
}
