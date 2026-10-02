package mangle

import (
	"reflect"
	"testing"
)

func TestCommands(t *testing.T) {
	rules := []Rule{
		{PodIP: "10.0.0.5", Class: "EF"},
		{PodIP: "10.0.0.2", Class: "AF11"},
		{PodIP: "10.0.0.9", Class: "not-a-class"},
	}

	cmds, skipped := Commands("DSCP_MARKER", rules)

	want := [][]string{
		{"-t", "mangle", "-F", "DSCP_MARKER"},
		{"-t", "mangle", "-A", "DSCP_MARKER", "-s", "10.0.0.2/32", "-j", "DSCP", "--set-dscp-class", "AF11"},
		{"-t", "mangle", "-A", "DSCP_MARKER", "-s", "10.0.0.5/32", "-j", "DSCP", "--set-dscp-class", "EF"},
	}

	if !reflect.DeepEqual(cmds, want) {
		t.Errorf("Commands() = %v, want %v", cmds, want)
	}
	if len(skipped) != 1 || skipped[0].PodIP != "10.0.0.9" {
		t.Errorf("Commands() skipped = %v, want one rule for 10.0.0.9", skipped)
	}
}

func TestCommandsEmpty(t *testing.T) {
	cmds, skipped := Commands("DSCP_MARKER", nil)
	want := [][]string{{"-t", "mangle", "-F", "DSCP_MARKER"}}
	if !reflect.DeepEqual(cmds, want) {
		t.Errorf("Commands(nil) = %v, want %v", cmds, want)
	}
	if len(skipped) != 0 {
		t.Errorf("Commands(nil) skipped = %v, want none", skipped)
	}
}

func TestIsValidClass(t *testing.T) {
	for _, c := range []string{"EF", "CS6", "AF11", "AF43", "default"} {
		if !IsValidClass(c) {
			t.Errorf("IsValidClass(%q) = false, want true", c)
		}
	}
	for _, c := range []string{"", "EF1", "af11", "CS8", "garbage"} {
		if IsValidClass(c) {
			t.Errorf("IsValidClass(%q) = true, want false", c)
		}
	}
}
