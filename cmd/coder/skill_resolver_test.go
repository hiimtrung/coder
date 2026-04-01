package main

import (
	"reflect"
	"testing"

	skilldomain "github.com/trungtran/coder/internal/domain/skill"
)

func TestNormalizeSkillNames(t *testing.T) {
	got := normalizeSkillNames(" architecture, golang ,Architecture,, testing ")
	want := []string{"architecture", "golang", "testing"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeSkillNames() = %v, want %v", got, want)
	}
}

func TestSelectResolvedSkills(t *testing.T) {
	results := []skilldomain.SkillSearchResult{
		{Skill: skilldomain.Skill{Name: "database"}, Score: 0.61},
		{Skill: skilldomain.Skill{Name: "golang"}, Score: 0.91},
		{Skill: skilldomain.Skill{Name: "architecture"}, Score: 0.82},
		{Skill: skilldomain.Skill{Name: "testing"}, Score: 0.55},
	}

	selected, keep, add, drop := selectResolvedSkills(results, []string{"database", "legacy"}, 3)

	if len(selected) != 3 {
		t.Fatalf("len(selected) = %d, want 3", len(selected))
	}

	gotSelected := []string{selected[0].Skill.Name, selected[1].Skill.Name, selected[2].Skill.Name}
	wantSelected := []string{"golang", "architecture", "database"}
	if !reflect.DeepEqual(gotSelected, wantSelected) {
		t.Fatalf("selected = %v, want %v", gotSelected, wantSelected)
	}

	if !reflect.DeepEqual(keep, []string{"database"}) {
		t.Fatalf("keep = %v, want %v", keep, []string{"database"})
	}
	if !reflect.DeepEqual(add, []string{"golang", "architecture"}) {
		t.Fatalf("add = %v, want %v", add, []string{"golang", "architecture"})
	}
	if !reflect.DeepEqual(drop, []string{"legacy"}) {
		t.Fatalf("drop = %v, want %v", drop, []string{"legacy"})
	}
}
