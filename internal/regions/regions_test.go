package regions

import (
	"slices"
	"testing"
)

func TestNames(t *testing.T) {
	names := Names()
	if !slices.IsSorted(names) || len(slices.Compact(slices.Clone(names))) != len(names) {
		t.Fatal("region list must be sorted and unique")
	}
	for _, name := range []string{"us-east-1", "eu-west-1", "ap-east-2", "ap-southeast-6", "mx-central-1", "cn-north-1", "us-gov-west-1", "eusc-de-east-1"} {
		if !Valid(name) {
			t.Errorf("missing region %q", name)
		}
	}
	for _, name := range []string{"", "not-a-region", "us-east-1a", "us-east-1; echo unsafe"} {
		if Valid(name) {
			t.Errorf("invalid region %q accepted", name)
		}
	}
}
