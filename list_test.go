package goreleases

import (
	"fmt"
	"testing"
)

func TestList(t *testing.T) {
	// disabled, don't bother real servers while testing
	if true {
		return
	}

	rels, err := ListSupported()
	if err != nil {
		t.Fatalf("listing supported releases: %s", err)
	}
	fmt.Println(rels)

	rels, err = ListAll()
	if err != nil {
		t.Fatalf("listing all releases: %s", err)
	}
	fmt.Println(rels)
}
