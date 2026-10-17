package config

import (
	"slices"
	"testing"
)

func TestCodingMenuEntriesHaveIndependentDetailsAndStartHidden(t *testing.T) {
	c := Default()
	for _, name := range []string{"git-branch", "git-sync", "runtimes", "dev-context", "last-command"} {
		if !slices.Contains(MenuItems(), name) || Shown(c.Visible, name) || len(DisplayParts(name)) < 2 {
			t.Fatalf("coding entry %q missing, shown, or lacks details", name)
		}
		found := false
		for _, entry := range c.OrderedMenuEntries() {
			if entry.Key == name {
				found = entry.Group == "CODING" && slices.Equal(entry.Parts, DisplayParts(name))
			}
		}
		if !found {
			t.Fatalf("%s is not in the CODING group", name)
		}
	}
	if !slices.Contains(c.MenuLayout.Groups, "CODING") || MenuSlots(c.Visible) != 11 {
		t.Fatal("coding group changed the default dashboard")
	}
	if err := c.ChangeDisplay("platform", "", false); err != nil {
		t.Fatal(err)
	}
	if err := c.ChangeDisplay("git-branch", "", true); err != nil {
		t.Fatal(err)
	}
	if err := c.ChangeDisplay("git-branch", "repository", false); err != nil {
		t.Fatal(err)
	}
	if !EntryShown(c.Visible, "git-branch") || Shown(c.Visible, "git-branch.repository") {
		t.Fatal("coding detail switch did not stay independent")
	}
}

func TestOldMenuLayoutsGainCodingGroup(t *testing.T) {
	c := Default()
	c.MenuLayout.Groups = []string{"NETWORK", "SYSTEM", "RESOURCES", "SESSION", "HEADER"}
	delete(c.MenuLayout.Items, "CODING")
	for _, name := range []string{"git-branch", "git-sync", "runtimes", "dev-context", "last-command"} {
		delete(c.MenuLayout.Components, name)
	}
	completed := completeMenuLayout(c.MenuLayout)
	if !slices.Equal(completed.Groups, []string{"NETWORK", "SYSTEM", "RESOURCES", "CODING", "SESSION", "HEADER"}) {
		t.Fatalf("migrated groups = %v", completed.Groups)
	}
	if err := validateMenuLayout(completed); err != nil {
		t.Fatal(err)
	}
}
