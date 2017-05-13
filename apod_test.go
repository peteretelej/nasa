package nasa

import (
	"strings"
	"testing"
)

func TestRandomAPOD(t *testing.T) {
	apod, err := RandomAPOD()
	if err != nil {
		t.Fatal(err)
	}
	if apod == nil {
		t.Errorf("invalid random apod, got <nil>")
	}
	if !strings.Contains(apod.String(), "Title:") {
		t.Errorf("invalid apod, not valid Image Stringer")
	}

}
