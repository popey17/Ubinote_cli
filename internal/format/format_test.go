package format_test

import (
	"testing"

	"ubinote-cli/internal/format"
)

func TestBoldWrapsSelection(t *testing.T) {
	r := format.Apply(format.Bold, "hello world", 6, 11)
	if r.Value != "hello **world**" {
		t.Fatalf("got %q", r.Value)
	}
	if r.SelectionStart != 8 || r.SelectionEnd != 13 {
		t.Fatalf("selection %d-%d", r.SelectionStart, r.SelectionEnd)
	}
}

func TestBoldPlaceholder(t *testing.T) {
	r := format.Apply(format.Bold, "", 0, 0)
	if r.Value != "**bold text**" {
		t.Fatalf("got %q", r.Value)
	}
}

func TestHeadingPrefixesLine(t *testing.T) {
	r := format.Apply(format.H2, "Title", 0, 5)
	if r.Value != "## Title" {
		t.Fatalf("got %q", r.Value)
	}
}

func TestULAndIndent(t *testing.T) {
	r := format.Apply(format.UL, "item", 0, 4)
	if r.Value != "- item" {
		t.Fatalf("got %q", r.Value)
	}
	r = format.Apply(format.Indent, r.Value, 0, len(r.Value))
	if r.Value != "    - item" {
		t.Fatalf("got %q", r.Value)
	}
}

func TestCheckAndCodeBlock(t *testing.T) {
	r := format.Apply(format.Check, "task", 0, 4)
	if r.Value != "- [ ] task" {
		t.Fatalf("got %q", r.Value)
	}
	r = format.Apply(format.CodeBlock, "x", 0, 1)
	if r.Value != "\n```\nx\n```\n" {
		t.Fatalf("got %q", r.Value)
	}
}

func TestLink(t *testing.T) {
	r := format.Apply(format.Link, "docs", 0, 4)
	if r.Value != "[docs](https://)" {
		t.Fatalf("got %q", r.Value)
	}
}
