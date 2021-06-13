// Copyright 2021 The Mellium Contributors.
// Use of this source code is governed by the BSD 2-clause
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"html/template"
	"image/png"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
)

const (
	dataURIPrefix = "data:image/png;base64,"
)

// A list of functions that will be added to loaded templates.
func tmplFuncs() template.FuncMap {
	return map[string]interface{}{
		"qr": barcoder(),
	}
}

func barcoder() func(template.URL) template.URL {
	buf := new(bytes.Buffer)

	return func(s template.URL) template.URL {
		buf.Reset()
		n, err := buf.WriteString(dataURIPrefix)
		if err != nil || n != len(dataURIPrefix) {
			panic(fmt.Sprintf("Error writing data URI, wrote %d bytes: %q", n, err))
		}
		qrCode, err := qr.Encode(string(s), qr.M, qr.Auto)
		if err != nil {
			panic(fmt.Sprintf("Error encoding QR Code: %q", err))
		}
		qrCode, err = barcode.Scale(qrCode, 200, 200)
		if err != nil {
			panic(fmt.Sprintf("Error scaling QR Code: %q", err))
		}
		e := base64.NewEncoder(base64.StdEncoding, buf)
		err = png.Encode(e, qrCode)
		if err != nil {
			panic(fmt.Sprintf("Error encoding QR Code PNG: %q", err))
		}
		/* #nosec */
		return template.URL(buf.String())
	}
}
