// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

//go:build !dev
// +build !dev

package main

import (
	"embed"
	"html/template"
)

var (
	//go:embed pages
	tmpls embed.FS

	//go:embed static
	assets embed.FS
)

var pagesGlob = "pages/*"

func reparseTemplates(tmpls *template.Template) (*template.Template, error) {
	return tmpls, nil
}
