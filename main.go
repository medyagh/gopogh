package main

import (
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/medyagh/gopogh/out"
	"github.com/medyagh/gopogh/parser"
)

var (
	inPath  = flag.String("in", "", "path to JSON input file")
	outPath = flag.String("out", "", "path to HTML output file")
	version = flag.Bool("version", true, "shows version")
)

func main() {
	flag.Parse()
	if *version {
		fmt.Printf("Version %s Build %s", out.Version, out.Build)
	}

	if *inPath == "" {
		panic("must provide path to JSON input file")
	}
	if *outPath == "" {
		panic("must provide path to HTML output file")
	}

	events, err := parser.ParseJSON(*inPath)
	if err != nil {
		panic(fmt.Sprintf("json: %v", err))
	}
	groups := parser.ProcessEvents(events)
	html, err := out.GenerateHTML(groups)
	if err != nil {
		panic(fmt.Sprintf("html: %v", err))
	}
	if err := ioutil.WriteFile(*outPath, html, 0644); err != nil {
		panic(fmt.Sprintf("write: %v", err))
	}
}
