package goreleases

import (
	"os"
	"runtime"
	"testing"
)

func TestFetch(t *testing.T) {
	// disabled, don't automatically bother production servers; especially for large downloads.
	if true {
		return
	}

	rels, err := ListSupported()
	if err != nil {
		t.Fatalf("fetching supported releases: %s", err)
	}

	rel := rels[0]
	file, err := FindFile(rel, runtime.GOOS, runtime.GOARCH, "archive")
	if err != nil {
		t.Fatalf("finding current hosts archive: %s", err)
	}
	t.Logf("fetching release %q", rel.Version)

	err = os.Mkdir("tmp", 0777)
	if err != nil {
		t.Fatalf("mkdir tmp: %s", err)
	}
	err = Fetch(file, "tmp", nil)
	if err != nil {
		t.Fatalf("fetch into tmp: %s", err)
	}
}
