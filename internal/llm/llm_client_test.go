package llm

import (
	"reflect"
	"testing"
)

func TestKeepKnownFilesDropsHallucinationsAndDuplicates(t *testing.T) {
	got := keepKnownFiles(
		[]string{"src/main.go", " missing.go ", "src/main.go", "README.md", ""},
		[]string{"README.md", "src/main.go"},
	)
	want := []string{"src/main.go", "README.md"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("known files = %#v, want %#v", got, want)
	}
}
