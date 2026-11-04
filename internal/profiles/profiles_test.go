package profiles

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestProfileAccountRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "profiles.conf")
	want := []Profile{{Name: "personal", KeyFile: "~/.ssh/personal", GHUser: "alice", GitName: "Alice", GitEmail: "alice@example.com", Sign: true, SignKey: "~/.ssh/sign.pub", OrgPattern: "^alice$"}, {Name: "legacy", KeyFile: "~/.ssh/work", OrgPattern: ".*"}}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip: got %#v, want %#v", got, want)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("permissions: %v", info.Mode())
	}
}
