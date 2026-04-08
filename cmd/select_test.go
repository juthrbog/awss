package cmd

import (
	"testing"

	"github.com/juthrbog/awss/internal/config"
)

func TestFormatExports_WithRegion(t *testing.T) {
	got := formatExports(config.Profile{Name: "production", Region: "us-east-1"}, "")
	want := "export AWS_PROFILE=production\nexport AWS_REGION=us-east-1\n"
	if got != want {
		t.Errorf("formatExports() =\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatExports_WithoutRegion(t *testing.T) {
	got := formatExports(config.Profile{Name: "staging", Region: ""}, "")
	want := "export AWS_PROFILE=staging\nunset AWS_REGION\n"
	if got != want {
		t.Errorf("formatExports() =\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatExports_DefaultProfile(t *testing.T) {
	got := formatExports(config.Profile{Name: "default", Region: "us-west-2"}, "")
	want := "export AWS_PROFILE=default\nexport AWS_REGION=us-west-2\n"
	if got != want {
		t.Errorf("formatExports() =\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatExports_Fish_WithRegion(t *testing.T) {
	got := formatExports(config.Profile{Name: "production", Region: "us-east-1"}, "fish")
	want := "set -gx AWS_PROFILE production\nset -gx AWS_REGION us-east-1\n"
	if got != want {
		t.Errorf("formatExports() =\n%s\nwant:\n%s", got, want)
	}
}

func TestFormatExports_Fish_WithoutRegion(t *testing.T) {
	got := formatExports(config.Profile{Name: "staging", Region: ""}, "fish")
	want := "set -gx AWS_PROFILE staging\nset -e AWS_REGION\n"
	if got != want {
		t.Errorf("formatExports() =\n%s\nwant:\n%s", got, want)
	}
}
