package main

import (
	"fmt"
	"github.com/unidoc/unipdf/v3/common/license"
	"github.com/unidoc/unipdf/v3/core"
	"github.com/unidoc/unipdf/v3/model"
	"os"
	"unsafe"
)

var WatermarkPath = []core.PdfObjectName{
	"PieceInfo",
	"ADBE_CompoundType",
	"Private",
}
const (
	WatermarkValue = "WatermarkDemo"
)

func find(d *core.PdfObjectDictionary, path []core.PdfObjectName, value core.PdfObjectName) bool {
	if len(path) == 0 {
		return false
	}
	key := path[0]

	if len(path) == 1 {
		elt, found := core.GetName(d.Get(key))
		return found && *elt == value
	}

	elt := d.Get(key)
	subDict, ok := core.GetDict(elt)
	if !ok || subDict == nil {
		return false
	}
	return find(subDict, path[1:], value)
}

func removeWatermark(p *model.PdfPage) bool {
	if p.Resources == nil || p.Resources.XObject == nil {
		return false
	}

	xObject, ok := core.GetDict(p.Resources.XObject)
	if !ok {
		return false
	}

	found := false
	for _, k := range xObject.Keys() {
		stream, ok := core.GetStream(xObject.Get(k))
		if !ok {
			continue
		}
		if !find(stream.PdfObjectDictionary, WatermarkPath, WatermarkValue) {
			continue
		}
		xObject.Remove(k)
		found = true
	}
	return found
}

//go:linkname lic github.com/unidoc/unipdf/v3/common/license.licenseKey
var lic uintptr

func init() {
	l := (*license.LicenseKey)(unsafe.Pointer(lic))
	l.Tier = license.LicenseTierCommunity
}

func fatalIf(err error, format string, values... interface{}) {
	if err == nil {
		return
	}
	msg := fmt.Sprintf(format, values...)
	fmt.Printf("FATAL, %s, %s\n", msg, err)
	os.Exit(1)
}

func loadPages(filename string) []*model.PdfPage {
	fmt.Printf("Reading %s...\n", filename)
	inFd, err := os.Open(filename)
	fatalIf(err, "failed to open input file")
	defer inFd.Close()

	pdfReader, err := model.NewPdfReader(inFd)
	fatalIf(err, "failed to decode %s", filename)

	return pdfReader.PageList
}

func main() {
	if len(os.Args) != 3 {
		fmt.Println(`Remove Master PDF Editor watermark from PDF

usage: pdf-no-wm input.pdf output.pdf
`)
		os.Exit(255)
	}

	writer := model.NewPdfWriter()

	for idx, page := range loadPages(os.Args[1]) {
		pageNumber := idx + 1
		found := removeWatermark(page)
		if found {
			fmt.Printf("Removed watermark from page %d\n", pageNumber)
		} else {
			fmt.Printf("No watermark found in page %d\n", pageNumber)
		}
		err := writer.AddPage(page)
		fatalIf(err, "failed to add page %d to output", pageNumber)
	}

	outFile := os.Args[2]
	outFd, err := os.Create(outFile)
	fatalIf(err, "failed to create output file")
	defer outFd.Close()

	err = writer.Write(outFd)
	fatalIf(err, "failed to create output PDF %s", outFile)
	fmt.Printf("Wrote output %q\n", outFile)
}