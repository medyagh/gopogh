package parser

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/medyagh/gopogh/pkg/models"
)

// ParseJSON is a very forgiving JSON parser.
func ParseJSON(path string) ([]models.TestEvent, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	events := []models.TestEvent{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		// Go's -json output is line-by-line JSON events
		b := scanner.Bytes()
		// Windows encodes its logs with nonsense \x00 characters, causing parsing to break entirely
		// stripping these characters away is harmless and fixes the issue.
		b = bytes.ReplaceAll(b, []byte("\x00"), []byte(""))
		if len(b) > 0 && b[0] == '{' {
			ev := models.TestEvent{}
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

// ProcessEvents group events by their test name
func ProcessEvents(evs []models.TestEvent) []models.TestGroup {
	gm := map[string]int{}
	groups := []models.TestGroup{}
	for _, e := range evs {
		if e.Test == "" {
			continue
		}
		index, ok := gm[e.Test]
		if !ok {
			index = len(groups)
			groups = append(groups, models.TestGroup{
				TestName: e.Test,
				Start:    e.Time,
			})
			gm[e.Test] = index
		}
		e.Output = strings.Trim(e.Output, " ")
		groups[index].Events = append(groups[index].Events, e)
		groups[index].Status = e.Action
	}

	// Hide ancestors
	for k, v := range gm {
		for k2 := range gm {
			if strings.HasPrefix(k2, fmt.Sprintf("%s/", k)) && !veto(v, groups) {
				groups[v].Hidden = true
			}
		}
	}

	return groups
}

// veto returns true is all child events pass, but parent event with pid fail.
// All other combinations will return false.
func veto(pid int, groups []models.TestGroup) bool {
	parent := groups[pid]
	if parent.Status != "fail" {
		return false
	}
	for _, child := range groups {
		if strings.HasPrefix(child.TestName, fmt.Sprintf("%s/", parent.TestName)) && child.Status != "pass" {
			return false
		}
	}
	return true
}
