package runtime

import (
	"bytes"
	"os"
	"testing"
)

func TestDependencySnapshot(t *testing.T) {
	for template, source := range map[string]string{"module.txt": "../../go.mod", "sums.txt": "../../go.sum"} {
		embedded, err := Source.ReadFile(template)
		if err != nil {
			t.Fatal(err)
		}
		actual, err := os.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(embedded, actual) {
			t.Fatalf("%s is stale; sync after go mod tidy", template)
		}
	}
}
