package skill

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDir(t *testing.T) {
	dir := t.TempDir()
	doc := `---
name: UI/UX Pro Max
description: Design intelligence for interfaces
version: 1.2.0
author: Xin
tags: [design, ui]
---

# UI/UX Pro Max

Use this skill for interface reviews.
`
	if err := os.WriteFile(filepath.Join(dir, DocName), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseDir(dir)
	if err != nil {
		t.Fatalf("ParseDir: %v", err)
	}
	if parsed.Meta.Name != "UI/UX Pro Max" || parsed.Meta.Version != "1.2.0" {
		t.Fatalf("unexpected metadata: %+v", parsed.Meta)
	}
	if parsed.Slug != "ui-ux-pro-max" {
		t.Fatalf("slug = %q", parsed.Slug)
	}
	if !strings.Contains(parsed.Body, "interface reviews") {
		t.Fatalf("body not parsed: %q", parsed.Body)
	}
}

func TestParseDocumentRejectsBadFrontmatter(t *testing.T) {
	cases := []string{
		"# no frontmatter",
		"---\nname: Missing close\n",
	}
	for _, input := range cases {
		if _, _, err := ParseDocument(input); err == nil {
			t.Fatalf("expected error for %q", input)
		}
	}
}

func TestSlugifyUnicodeFallback(t *testing.T) {
	if Slugify("My Skill!") != "my-skill" {
		t.Fatalf("ascii slug failed: %q", Slugify("My Skill!"))
	}
	slug := Slugify("中文技能")
	if !strings.HasPrefix(slug, "skill-") {
		t.Fatalf("unicode fallback = %q", slug)
	}
}
