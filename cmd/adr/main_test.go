package main

import (
	"reflect"
	"testing"
)

func TestParseGlobalOptions(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantGlobal  globalOptions
		wantCommand string
		wantArgs    []string
	}{
		{
			name:        "options before command",
			args:        []string{"--dir", "custom/adr", "--format", "json", "list", "--status", "adopted"},
			wantGlobal:  globalOptions{Dir: "custom/adr", Format: "json"},
			wantCommand: "list",
			wantArgs:    []string{"--status", "adopted"},
		},
		{
			name:        "json shorthand",
			args:        []string{"--json", "review", "--base", "main"},
			wantGlobal:  globalOptions{Format: "json"},
			wantCommand: "review",
			wantArgs:    []string{"--base", "main"},
		},
		{
			name:        "double dash",
			args:        []string{"--dir=custom/adr", "--", "show", "0001"},
			wantGlobal:  globalOptions{Dir: "custom/adr"},
			wantCommand: "show",
			wantArgs:    []string{"0001"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			global, command, args, err := parseGlobalOptions(tt.args)
			if err != nil {
				t.Fatalf("parseGlobalOptions() error = %v", err)
			}
			if !reflect.DeepEqual(global, tt.wantGlobal) {
				t.Errorf("global = %#v, want %#v", global, tt.wantGlobal)
			}
			if command != tt.wantCommand {
				t.Errorf("command = %q, want %q", command, tt.wantCommand)
			}
			if !reflect.DeepEqual(args, tt.wantArgs) {
				t.Errorf("args = %#v, want %#v", args, tt.wantArgs)
			}
		})
	}
}

func TestStringListValues(t *testing.T) {
	var values stringListValues
	for _, value := range []string{"go, rust", "cli", " , "} {
		if err := values.Set(value); err != nil {
			t.Fatalf("Set(%q) error = %v", value, err)
		}
	}

	want := stringListValues{"go", "rust", "cli"}
	if !reflect.DeepEqual(values, want) {
		t.Errorf("values = %#v, want %#v", values, want)
	}
}
