package config

import "testing"

func TestNormalizeResourceSetMountMode(t *testing.T) {
	set := &ResourceSet{
		Mounts: []Mount{
			{Source: "/tmp", Target: "/mnt", Mode: "RW"},
		},
	}
	if err := NormalizeResourceSet(set, "test"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := set.Mounts[0].Mode; got != "rw" {
		t.Fatalf("expected mount mode normalized to rw, got %q", got)
	}
}

func TestNormalizeResourceSetInvalidMountMode(t *testing.T) {
	set := &ResourceSet{
		Mounts: []Mount{
			{Source: "/tmp", Target: "/mnt", Mode: "bad"},
		},
	}
	if err := NormalizeResourceSet(set, "test"); err == nil {
		t.Fatalf("expected error for invalid mount mode")
	}
}

func TestNormalizeResourceSetCallValidation(t *testing.T) {
	set := &ResourceSet{
		Calls: []Call{
			{Name: "", Command: "echo hi"},
		},
	}
	if err := NormalizeResourceSet(set, "test"); err == nil {
		t.Fatalf("expected error for missing call name")
	}

	set = &ResourceSet{
		Calls: []Call{
			{Name: "do", Command: ""},
		},
	}
	if err := NormalizeResourceSet(set, "test"); err == nil {
		t.Fatalf("expected error for missing call command")
	}
}

func TestNormalizeResourceSetAllowedArgsRegex(t *testing.T) {
	set := &ResourceSet{
		Calls: []Call{
			{Name: "do", Command: "echo hi", AllowedArgs: "("},
		},
	}
	if err := NormalizeResourceSet(set, "test"); err == nil {
		t.Fatalf("expected error for invalid allowed-args regex")
	}
}
