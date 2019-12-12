package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/template"
	"time"

	rice "github.com/GeertJohan/go.rice"
)

var (
	inPath  = flag.String("in", "", "path to JSON input file")
	outPath = flag.String("out", "", "path to HTML output file")
)

type TestEvent struct {
	Time    time.Time // encodes as an RFC3339-format string
	Action  string
	Package string
	Test    string
	Elapsed float64 // seconds
	Output  string

	EmbeddedLog []string
}

type TestGroup struct {
	Test     string
	Hidden   bool
	Status   string
	Start    time.Time
	End      time.Time
	Duration float64
	Events   []TestEvent
}

// parseJSON is a very forgiving JSON parser.
func parseJSON(path string) ([]TestEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	events := []TestEvent{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		// Go's -json output is line-by-line JSON events
		b := scanner.Bytes()
		if b[0] == '{' {
			ev := TestEvent{}
			err = json.Unmarshal(b, &ev)
			if err != nil {
				continue
			}
			events = append(events, ev)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return events, err
}

// group events by their test name
func processEvents(evs []TestEvent) []TestGroup {
	gm := map[string]int{}
	groups := []TestGroup{}
	for _, e := range evs {
		if e.Test == "" {
			continue
		}
		index, ok := gm[e.Test]
		if !ok {
			index = len(groups)
			groups = append(groups, TestGroup{
				Test:  e.Test,
				Start: e.Time,
			})
			gm[e.Test] = index
		}
		groups[index].Events = append(groups[index].Events, e)
		groups[index].Status = e.Action
	}

	// Hide ancestors
	for k, v := range gm {
		for k2 := range gm {
			if strings.HasPrefix(k2, fmt.Sprintf("%s/", k)) {
				groups[v].Hidden = true
			}
		}
	}

	return groups
}

func generateHTML(groups []TestGroup) ([]byte, error) {
	for _, g := range groups {
		g.Duration = g.Events[len(g.Events)-1].Elapsed
	}
	t, err := template.New("out").Parse(rice.MustFindBox("template").MustString("report.html"))
	if err != nil {
		return nil, err
	}

	type content struct {
		TestName string
		Duration string
		Result   string
		Events   string
	}

	var contents []content
	for _, g := range groups {
		d := fmt.Sprintf("%f", g.Events[len(g.Events)-1].Elapsed)
		if !g.Hidden {
			contents = append(contents, content{TestName: g.Test, Duration: d, Result: g.Status})
		}
	}

	json, err := json.Marshal(contents)
	if err != nil {
		log.Fatalf("json marshal %v", json)
	}

	if err := ioutil.WriteFile("j.json", json, 0644); err != nil {
		panic(fmt.Sprintf("write: %v", err))
	}

	var b bytes.Buffer
	if err := t.ExecuteTemplate(&b, "out", string(json)); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func main() {
	flag.Parse()
	if *inPath == "" {
		panic("must provide path to JSON input file")
	}
	if *outPath == "" {
		panic("must provide path to HTML output file")
	}

	events, err := parseJSON(*inPath)
	if err != nil {
		panic(fmt.Sprintf("json: %v", err))
	}
	groups := processEvents(events)
	html, err := generateHTML(groups)
	if err != nil {
		panic(fmt.Sprintf("html: %v", err))
	}
	if err := ioutil.WriteFile(*outPath, html, 0644); err != nil {
		panic(fmt.Sprintf("write: %v", err))
	}
}
