package format

import (
	"regexp"
	"strings"
)

type Action string

const (
	Bold      Action = "bold"
	Italic    Action = "italic"
	H1        Action = "h1"
	H2        Action = "h2"
	H3        Action = "h3"
	UL        Action = "ul"
	OL        Action = "ol"
	Check     Action = "check"
	Quote     Action = "quote"
	Code      Action = "code"
	CodeBlock Action = "codeblock"
	Link      Action = "link"
	Indent    Action = "indent"
	Outdent   Action = "outdent"
)

type Result struct {
	Value          string
	SelectionStart int
	SelectionEnd   int
}

// CommonMark needs ≥3 spaces to nest under `1. `; 4 is safe for ol + ul.
const indentUnit = "    "

var (
	listMarkerRe = regexp.MustCompile(`^([-*+]|\d+\.)\s+`)
	taskMarkerRe = regexp.MustCompile(`^\[[ xX]\]\s+`)
	headingRe    = regexp.MustCompile(`^#{1,6}\s+`)
	quoteRe      = regexp.MustCompile(`^>\s?`)
)

func AllActions() []Action {
	return []Action{
		H1, H2, H3, Bold, Italic, UL, OL, Indent, Outdent,
		Check, Quote, Code, CodeBlock, Link,
	}
}

func ActionLabel(a Action) string {
	switch a {
	case H1:
		return "Heading 1"
	case H2:
		return "Heading 2"
	case H3:
		return "Heading 3"
	case Bold:
		return "Bold"
	case Italic:
		return "Italic"
	case UL:
		return "Bullet list"
	case OL:
		return "Numbered list"
	case Indent:
		return "Indent"
	case Outdent:
		return "Outdent"
	case Check:
		return "Checklist"
	case Quote:
		return "Quote"
	case Code:
		return "Inline code"
	case CodeBlock:
		return "Code block"
	case Link:
		return "Link"
	default:
		return string(a)
	}
}

func Apply(action Action, value string, start, end int) Result {
	if start < 0 {
		start = 0
	}
	if end < start {
		end = start
	}
	if end > len(value) {
		end = len(value)
	}
	if start > len(value) {
		start = len(value)
	}

	switch action {
	case Bold:
		return wrapInline(value, start, end, "**", "**", "bold text")
	case Italic:
		return wrapInline(value, start, end, "*", "*", "italic text")
	case Code:
		return wrapInline(value, start, end, "`", "`", "code")
	case Link:
		return wrapInline(value, start, end, "[", "](https://)", "link text")
	case H1:
		return prefixLines(value, start, end, func(_ int, ind, rest string) string {
			return ind + "# " + headingRe.ReplaceAllString(rest, "")
		}, "List item")
	case H2:
		return prefixLines(value, start, end, func(_ int, ind, rest string) string {
			return ind + "## " + headingRe.ReplaceAllString(rest, "")
		}, "List item")
	case H3:
		return prefixLines(value, start, end, func(_ int, ind, rest string) string {
			return ind + "### " + headingRe.ReplaceAllString(rest, "")
		}, "List item")
	case UL:
		return prefixLines(value, start, end, func(_ int, ind, rest string) string {
			return ind + "- " + stripListMarker(rest)
		}, "List item")
	case OL:
		return prefixLines(value, start, end, func(i int, ind, rest string) string {
			return ind + itoa(i+1) + ". " + stripListMarker(rest)
		}, "List item")
	case Check:
		return prefixLines(value, start, end, func(_ int, ind, rest string) string {
			return ind + "- [ ] " + stripListMarker(rest)
		}, "List item")
	case Quote:
		return prefixLines(value, start, end, func(_ int, ind, rest string) string {
			return ind + "> " + quoteRe.ReplaceAllString(rest, "")
		}, "quote")
	case Indent:
		return changeIndent(value, start, end, "in")
	case Outdent:
		return changeIndent(value, start, end, "out")
	case CodeBlock:
		selected := value[start:end]
		if selected == "" {
			selected = "code"
		}
		block := "\n```\n" + selected + "\n```\n"
		next := value[:start] + block + value[end:]
		innerStart := start + len("\n```\n")
		return Result{
			Value:          next,
			SelectionStart: innerStart,
			SelectionEnd:   innerStart + len(selected),
		}
	default:
		return Result{Value: value, SelectionStart: start, SelectionEnd: end}
	}
}

func wrapInline(value string, start, end int, before, after, placeholder string) Result {
	selected := value[start:end]
	content := selected
	if content == "" {
		content = placeholder
	}
	next := value[:start] + before + content + after + value[end:]
	selStart := start + len(before)
	return Result{
		Value:          next,
		SelectionStart: selStart,
		SelectionEnd:   selStart + len(content),
	}
}

func selectedLines(value string, start, end int) (lineStart, lineEnd int, lines []string) {
	lineStart = strings.LastIndex(value[:start], "\n") + 1
	lineEnd = strings.Index(value[end:], "\n")
	if lineEnd == -1 {
		lineEnd = len(value)
	} else {
		lineEnd = end + lineEnd
	}
	block := value[lineStart:lineEnd]
	if len(block) == 0 {
		return lineStart, lineEnd, []string{""}
	}
	return lineStart, lineEnd, strings.Split(block, "\n")
}

func replaceLines(value string, lineStart, lineEnd int, lines []string) Result {
	formatted := strings.Join(lines, "\n")
	next := value[:lineStart] + formatted + value[lineEnd:]
	return Result{
		Value:          next,
		SelectionStart: lineStart,
		SelectionEnd:   lineStart + len(formatted),
	}
}

func stripListMarker(text string) string {
	text = listMarkerRe.ReplaceAllString(text, "")
	text = taskMarkerRe.ReplaceAllString(text, "")
	return text
}

func splitIndent(line string) (indent, rest string) {
	i := 0
	for i < len(line) && (line[i] == ' ' || line[i] == '\t') {
		i++
	}
	return line[:i], line[i:]
}

func prefixLines(
	value string,
	start, end int,
	prefixFor func(index int, indent, rest string) string,
	emptyPlaceholder string,
) Result {
	lineStart, lineEnd, lines := selectedLines(value, start, end)
	source := lines
	if len(lines) == 1 && lines[0] == "" {
		source = []string{emptyPlaceholder}
	}
	formatted := make([]string, len(source))
	for i, line := range source {
		raw := line
		if len(raw) == 0 {
			raw = emptyPlaceholder
		}
		ind, rest := splitIndent(raw)
		formatted[i] = prefixFor(i, ind, rest)
	}
	return replaceLines(value, lineStart, lineEnd, formatted)
}

func changeIndent(value string, start, end int, direction string) Result {
	lineStart, lineEnd, lines := selectedLines(value, start, end)
	nextLines := make([]string, len(lines))
	for i, line := range lines {
		if direction == "in" {
			nextLines[i] = indentUnit + line
			continue
		}
		if strings.HasPrefix(line, indentUnit) {
			nextLines[i] = line[len(indentUnit):]
			continue
		}
		if strings.HasPrefix(line, "\t") {
			nextLines[i] = line[1:]
			continue
		}
		trimmed := line
		n := 0
		for n < len(trimmed) && n < 4 && trimmed[n] == ' ' {
			n++
		}
		nextLines[i] = trimmed[n:]
	}
	return replaceLines(value, lineStart, lineEnd, nextLines)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
