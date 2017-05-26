package nasa

import (
	"strings"
	"testing"
	"time"
)

func TestNeoFeed(t *testing.T) {
	type neotests struct {
		startText, endText string
		nl                 *NeoList
		err                error
	}
	var tests = []neotests{
		{
			startText: "2017-05-11",
			endText:   "2017-05-12",
			nl:        &NeoList{ElementCount: 18},
		},
		{
			startText: "1000-05-11",
			endText:   "1000-05-12",
			nl:        &NeoList{ElementCount: 0},
		},
	}

	for _, v := range tests {
		start, err := time.Parse("2006-01-02", v.startText)
		if err != nil {
			t.Fatal(err)
		}
		end, err := time.Parse("2006-01-02", v.endText)
		if err != nil {
			t.Fatal(err)
		}
		nl, err := NeoFeed(start, end)
		if err != v.err {
			t.Errorf("NeoFeed returned wrong error got %v, want %v", err, v.err)
		}
		if nl == nil {
			if v.nl != nil {
				t.Errorf("NeoFeed returned NeoList when %v was expected", nil)
			}
			continue
		}
		if nl.ElementCount != v.nl.ElementCount {
			t.Errorf("NeoFeed returned wrong element count got %d, want %d", nl.ElementCount, v.nl.ElementCount)
		}
		if !strings.Contains(nl.String(), "Near Earth Objects From:") {
			t.Errorf("NeoFeed return an invalid neolist, not valid NeoList stringer")
		}
	}
}
