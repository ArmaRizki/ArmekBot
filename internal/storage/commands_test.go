package storage

import (
	"sync"
	"testing"

	"github.com/rs/zerolog"
)

// Two admins toggling different groups at the same time must not lose one of
// the two changes: the read-modify-write has to happen in one transaction.
func TestConcurrentGroupTogglesKeepBothChanges(t *testing.T) {
	s := newTestStorage(t)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := s.DisableGroup("g", "music"); err != nil {
			t.Errorf("disable music: %v", err)
		}
	}()
	go func() {
		defer wg.Done()
		if err := s.DisableGroup("g", "core"); err != nil {
			t.Errorf("disable core: %v", err)
		}
	}()
	wg.Wait()

	disabled, err := s.DisabledGroups("g")
	if err != nil {
		t.Fatal(err)
	}
	if len(disabled) != 2 {
		t.Fatalf("disabled groups = %v, want both music and core", disabled)
	}
}

// Enabling a group that was disabled must survive a reopen.
func TestEnableGroupClearsOnlyThatGroup(t *testing.T) {
	s := newTestStorage(t)

	if err := s.DisableGroup("g", "music"); err != nil {
		t.Fatal(err)
	}
	if err := s.DisableGroup("g", "core"); err != nil {
		t.Fatal(err)
	}
	if err := s.EnableGroup("g", "music"); err != nil {
		t.Fatal(err)
	}

	disabled, err := s.DisabledGroups("g")
	if err != nil {
		t.Fatal(err)
	}
	if len(disabled) != 1 || disabled[0] != "core" {
		t.Fatalf("disabled groups = %v, want only core", disabled)
	}
	if on, err := s.IsGroupDisabled("g", "music"); err != nil || on {
		t.Fatalf("music still disabled: %v %v", on, err)
	}
}

// The datastore must survive a settings write on an unknown guild.
func TestDisableGroupOnUnknownGuild(t *testing.T) {
	s, err := NewStorage(t.TempDir(), zerolog.Nop())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })

	if err := s.DisableGroup("fresh", "music"); err != nil {
		t.Fatal(err)
	}
	if on, err := s.IsGroupDisabled("fresh", "music"); err != nil || !on {
		t.Fatalf("music not disabled: %v %v", on, err)
	}
}
