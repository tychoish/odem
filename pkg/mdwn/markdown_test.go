package mdwn

import (
	"iter"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/tychoish/fun/irt"
)

// mb is a test helper that builds content and returns the string.
func build(fn func(*Builder)) string {
	var mb Builder
	fn(&mb)
	return mb.String()
}

// --- Headings ---

func TestH1(t *testing.T) {
	got := build(func(m *Builder) { m.H1("Title") })
	want := "# Title\n\n"
	if got != want {
		t.Errorf("H1: got %q, want %q", got, want)
	}
}

func TestH2(t *testing.T) {
	got := build(func(m *Builder) { m.H2("Section") })
	want := "## Section\n\n"
	if got != want {
		t.Errorf("H2: got %q, want %q", got, want)
	}
}

func TestH3(t *testing.T) {
	got := build(func(m *Builder) { m.H3("Sub") })
	want := "### Sub\n\n"
	if got != want {
		t.Errorf("H3: got %q, want %q", got, want)
	}
}

// --- Paragraph ---

func TestParagraph(t *testing.T) {
	got := build(func(m *Builder) { m.Paragraph("Hello world.") })
	if got != "Hello world.\n\n" {
		t.Errorf("Paragraph: got %q", got)
	}
}

func TestItalicParagraph(t *testing.T) {
	got := build(func(m *Builder) { m.ItalicParagraph("note") })
	if got != "_note_\n\n" {
		t.Errorf("ItalicParagraph: got %q", got)
	}
}

// --- KV ---

func TestKV(t *testing.T) {
	got := build(func(m *Builder) { m.KV("Name", "Alice") })
	if got != "**Name**: Alice\n" {
		t.Errorf("KV: got %q", got)
	}
}

// --- Lists ---

func TestBulletListEmpty(t *testing.T) {
	got := build(func(m *Builder) { m.BulletList() })
	if got != "" {
		t.Errorf("BulletList(empty): expected empty string, got %q", got)
	}
}

func TestBulletList(t *testing.T) {
	got := build(func(m *Builder) { m.BulletList("alpha", "beta", "gamma") })
	want := "- alpha\n- beta\n- gamma\n\n"
	if got != want {
		t.Errorf("BulletList: got %q, want %q", got, want)
	}
}

func TestBulletListItem(t *testing.T) {
	got := build(func(m *Builder) {
		m.BulletListItem("one")
		m.BulletListItem("two")
	})
	if got != "- one\n- two\n" {
		t.Errorf("BulletListItem: got %q", got)
	}
}

func TestExtendBulletList(t *testing.T) {
	items := []string{"x", "y", "z"}
	got := build(func(m *Builder) {
		m.ExtendBulletList(func(yield func(string) bool) {
			for _, s := range items {
				if !yield(s) {
					return
				}
			}
		})
	})
	want := "- x\n- y\n- z\n\n"
	if got != want {
		t.Errorf("ExtendBulletList: got %q, want %q", got, want)
	}
}

func TestExtendBulletListEmpty(t *testing.T) {
	got := build(func(m *Builder) {
		m.ExtendBulletList(func(yield func(string) bool) {})
	})
	if got != "" {
		t.Errorf("ExtendBulletList(empty): expected empty, got %q", got)
	}
}

func TestOrderedListEmpty(t *testing.T) {
	got := build(func(m *Builder) { m.OrderedList() })
	if got != "" {
		t.Errorf("OrderedList(empty): expected empty string, got %q", got)
	}
}

func TestOrderedList(t *testing.T) {
	got := build(func(m *Builder) { m.OrderedList("first", "second", "third") })
	want := "1. first\n1. second\n1. third\n\n"
	if got != want {
		t.Errorf("OrderedList: got %q, want %q", got, want)
	}
}

func TestOrderedListItem(t *testing.T) {
	got := build(func(m *Builder) {
		m.OrderedListItem("a")
		m.OrderedListItem("b")
	})
	if got != "1. a\n1. b\n" {
		t.Errorf("OrderedListItem: got %q", got)
	}
}

// --- Block quote ---

func TestBlockQuote(t *testing.T) {
	got := build(func(m *Builder) { m.BlockQuote("line one\nline two") })
	want := "> line one\n> line two\n\n"
	if got != want {
		t.Errorf("BlockQuote: got %q, want %q", got, want)
	}
}

func TestBlockQuoteTrailingNewline(t *testing.T) {
	// Trailing newlines in the input should not produce extra "> " lines.
	got := build(func(m *Builder) { m.BlockQuote("text\n\n") })
	want := "> text\n\n"
	if got != want {
		t.Errorf("BlockQuote(trailing newline): got %q, want %q", got, want)
	}
}

func TestBlockQuotePreservesBlankLines(t *testing.T) {
	// Blank lines in the middle of the text must be preserved.
	got := build(func(m *Builder) { m.BlockQuote("para one\n\npara two") })
	want := "> para one\n> \n> para two\n\n"
	if got != want {
		t.Errorf("BlockQuote(blank line): got %q, want %q", got, want)
	}
}

func TestBlockQuoteWith(t *testing.T) {
	got := build(func(m *Builder) {
		m.BlockQuoteWith(func(inner *Builder) {
			inner.BulletList("item a", "item b")
		})
	})
	// Each line of the inner output should be prefixed with "> ".
	for line := range strings.SplitSeq(strings.TrimRight(got, "\n"), "\n") {
		if line != "" && !strings.HasPrefix(line, "> ") {
			t.Errorf("BlockQuoteWith: line %q missing '> ' prefix", line)
		}
	}
	if !strings.Contains(got, "> - item a") {
		t.Errorf("BlockQuoteWith: expected '> - item a' in output, got %q", got)
	}
}

// --- Fenced code ---

func TestFencedCode(t *testing.T) {
	got := build(func(m *Builder) { m.FencedCode("go", "fmt.Println()") })
	want := "```go\nfmt.Println()\n```\n\n"
	if got != want {
		t.Errorf("FencedCode: got %q, want %q", got, want)
	}
}

func TestFencedCodeNoLang(t *testing.T) {
	got := build(func(m *Builder) { m.FencedCode("", "x := 1\n") })
	want := "```\nx := 1\n```\n\n"
	if got != want {
		t.Errorf("FencedCode(no lang): got %q, want %q", got, want)
	}
}

// --- Inline formatters ---

func TestBold(t *testing.T) {
	got := build(func(m *Builder) { m.Bold("important") })
	if got != "**important**" {
		t.Errorf("Bold: got %q", got)
	}
}

func TestItalic(t *testing.T) {
	got := build(func(m *Builder) { m.Italic("em") })
	if got != "_em_" {
		t.Errorf("Italic: got %q", got)
	}
}

func TestPreformatted(t *testing.T) {
	got := build(func(m *Builder) { m.Preformatted("code") })
	if got != "`code`" {
		t.Errorf("Preformatted: got %q", got)
	}
}

func TestLink(t *testing.T) {
	got := build(func(m *Builder) { m.Link("click", "https://example.com") })
	if got != "[click](https://example.com)" {
		t.Errorf("Link: got %q", got)
	}
}

func TestStrikethrough(t *testing.T) {
	got := build(func(m *Builder) { m.Strikethrough("old") })
	if got != "~~old~~" {
		t.Errorf("Strikethrough: got %q", got)
	}
}

func TestInlineChaining(t *testing.T) {
	got := build(func(m *Builder) {
		m.Bold("Note").Text(": see ").Link("docs", "https://example.com").ParagraphBreak()
	})
	want := "**Note**: see [docs](https://example.com)\n\n"
	if got != want {
		t.Errorf("inline chaining: got %q, want %q", got, want)
	}
}

// --- ParagraphBreak ---

func TestParagraphBreak(t *testing.T) {
	got := build(func(m *Builder) {
		m.PushString("a")
		m.ParagraphBreak()
		m.PushString("b")
	})
	if got != "a\n\nb" {
		t.Errorf("ParagraphBreak: got %q", got)
	}
}

// --- TableBuilder ---

func TestTableBasic(t *testing.T) {
	got := build(func(m *Builder) {
		m.NewTable(
			Column{Name: "Name"},
			Column{Name: "Count", RightAlign: true},
		).Row("Alice", "42").Row("Bob", "7").Build()
	})

	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) < 4 { // header + separator + 2 rows (blank line stripped)
		t.Fatalf("TableBasic: expected at least 4 lines, got %d:\n%s", len(lines), got)
	}
	if !strings.HasPrefix(lines[0], "| Name") {
		t.Errorf("TableBasic: header line = %q", lines[0])
	}
	if !strings.Contains(lines[1], "---") {
		t.Errorf("TableBasic: separator line = %q", lines[1])
	}
	// Right-aligned separator ends with ":".
	if !strings.Contains(lines[1], ":") {
		t.Errorf("TableBasic: right-align separator missing colon: %q", lines[1])
	}
	if !strings.Contains(lines[2], "Alice") {
		t.Errorf("TableBasic: data row 0 = %q", lines[2])
	}
	if !strings.Contains(lines[3], "Bob") {
		t.Errorf("TableBasic: data row 1 = %q", lines[3])
	}
}

func TestTablePipeEscaping(t *testing.T) {
	got := build(func(m *Builder) {
		m.NewTable(Column{Name: "Val"}).Row("a|b").Build()
	})
	if !strings.Contains(got, `a\|b`) {
		t.Errorf("TablePipeEscaping: pipe not escaped in %q", got)
	}
}

func TestTableColumnAlignment(t *testing.T) {
	got := build(func(m *Builder) {
		m.NewTable(
			Column{Name: "L"},
			Column{Name: "R", RightAlign: true},
		).Row("left", "1234").Build()
	})
	lines := strings.Split(got, "\n")
	dataRow := lines[2]
	// In a right-aligned column "1234" should be flush-right; the left cell "left"
	// should be flush-left. Both cells are space-padded to column width.
	if !strings.Contains(dataRow, "| left") {
		t.Errorf("TableAlignment: left cell not left-aligned in %q", dataRow)
	}
	if !strings.Contains(dataRow, "1234 |") {
		t.Errorf("TableAlignment: right cell not right-aligned in %q", dataRow)
	}
}

func TestTableMinWidth(t *testing.T) {
	got := build(func(m *Builder) {
		m.NewTable(Column{Name: "X", MinWidth: 10}).Row("hi").Build()
	})
	// Column should be at least 10 chars wide (excluding padding pipes).
	lines := strings.Split(got, "\n")
	header := lines[0]
	// Count characters between the first and second "|".
	parts := strings.Split(header, "|")
	if len(parts) < 2 {
		t.Fatalf("TableMinWidth: unexpected header %q", header)
	}
	cellWidth := len(parts[1]) - 2 // subtract the two spaces
	if cellWidth < 10 {
		t.Errorf("TableMinWidth: column width %d < MinWidth 10", cellWidth)
	}
}

func TestTableMaxWidth(t *testing.T) {
	got := build(func(m *Builder) {
		m.NewTable(Column{Name: "T", MaxWidth: 8}).Row("short").Row("this is a very long value").Build()
	})
	lines := strings.Split(got, "\n")
	longRow := lines[3] // header, sep, short, long
	if strings.Contains(longRow, "this is a very long value") {
		t.Errorf("TableMaxWidth: long value was not truncated: %q", longRow)
	}
	if !strings.Contains(longRow, "...") {
		t.Errorf("TableMaxWidth: truncated cell missing ellipsis: %q", longRow)
	}
}

func TestTableRows(t *testing.T) {
	rows := [][]string{{"a", "1"}, {"b", "2"}}
	got := build(func(m *Builder) {
		m.NewTable(Column{Name: "K"}, Column{Name: "V"}).Rows(rows).Build()
	})
	if !strings.Contains(got, "| a") || !strings.Contains(got, "| b") {
		t.Errorf("TableRows: expected rows a,b in %q", got)
	}
}

func TestTableExtend(t *testing.T) {
	seq := func(yield func([]string) bool) {
		for _, row := range [][]string{{"x", "10"}, {"y", "20"}} {
			if !yield(row) {
				return
			}
		}
	}
	got := build(func(m *Builder) {
		m.NewTable(Column{Name: "K"}, Column{Name: "V"}).
			Extend(iter.Seq[[]string](seq)).Build()
	})
	if !strings.Contains(got, "| x") || !strings.Contains(got, "| y") {
		t.Errorf("TableExtend: expected rows x,y in %q", got)
	}
}

func TestTableEmptyRows(t *testing.T) {
	got := build(func(m *Builder) {
		m.NewTable(Column{Name: "A"}, Column{Name: "B"}).Build()
	})
	// Build returns early when there are no rows, producing no output.
	if got != "" {
		t.Errorf("TableEmptyRows: expected empty output, got %q", got)
	}
}

func TestTableEndsWithBlankLine(t *testing.T) {
	got := build(func(m *Builder) {
		m.NewTable(Column{Name: "X"}).Row("v").Build()
	})
	if !strings.HasSuffix(got, "\n\n") {
		t.Errorf("Table: expected trailing blank line, got %q", got)
	}
}

func TestExtendOrderedList(t *testing.T) {
	items := []string{"one", "two", "three"}
	got := build(func(m *Builder) {
		m.ExtendOrderedList(func(yield func(string) bool) {
			for _, s := range items {
				if !yield(s) {
					return
				}
			}
		})
	})
	want := "1. one\n1. two\n1. three\n\n"
	if got != want {
		t.Errorf("ExtendOrderedList: got %q, want %q", got, want)
	}
}

func TestExtendOrderedListEmpty(t *testing.T) {
	got := build(func(m *Builder) {
		m.ExtendOrderedList(func(yield func(string) bool) {})
	})
	if got != "" {
		t.Errorf("ExtendOrderedList(empty): expected empty, got %q", got)
	}
}

func TestBlockQuoteWithEmpty(t *testing.T) {
	got := build(func(m *Builder) { m.BlockQuoteWith(func(*Builder) {}) })
	if got != "" {
		t.Errorf("BlockQuoteWith(empty fn): expected empty output, got %q", got)
	}
}

func TestBlockQuoteEmpty(t *testing.T) {
	got := build(func(m *Builder) { m.BlockQuote("") })
	if got != "" {
		t.Errorf("BlockQuote(empty): expected empty, got %q", got)
	}
}

func TestBlockQuoteOnlyNewlines(t *testing.T) {
	got := build(func(m *Builder) { m.BlockQuote("\n\n") })
	if got != "" {
		t.Errorf("BlockQuote(only newlines): expected empty, got %q", got)
	}
}

func TestTableExtendRow(t *testing.T) {
	seq := func(yield func(string) bool) {
		for _, s := range []string{"alpha", "99"} {
			if !yield(s) {
				return
			}
		}
	}
	got := build(func(m *Builder) {
		m.NewTable(Column{Name: "K"}, Column{Name: "V"}).
			ExtendRow(iter.Seq[string](seq)).Build()
	})
	if !strings.Contains(got, "alpha") || !strings.Contains(got, "99") {
		t.Errorf("TableExtendRow: expected row in output, got %q", got)
	}
}

func TestTableCustomTruncMarker(t *testing.T) {
	got := build(func(m *Builder) {
		m.NewTable(Column{Name: "T", MaxWidth: 10, TruncMarker: "…"}).
			Row("this is definitely longer than ten characters").Build()
	})
	if !strings.Contains(got, "…") {
		t.Errorf("TableCustomTruncMarker: expected custom marker in %q", got)
	}
}

func TestTableNarrowTruncation(t *testing.T) {
	// MaxWidth=3 with default marker "..." (len=3): widths[i] is not > len(marker),
	// so the else branch truncates by slicing without appending the marker.
	got := build(func(m *Builder) {
		m.NewTable(Column{Name: "X", MaxWidth: 3}).Row("hello").Build()
	})
	lines := strings.Split(got, "\n")
	dataRow := lines[2]
	if strings.Contains(dataRow, "hello") {
		t.Errorf("TableNarrowTruncation: expected cell truncated, got %q", dataRow)
	}
}

func TestKVTable(t *testing.T) {
	got := build(func(m *Builder) {
		m.KVTable(
			irt.MakeKV("Name", "Count"),
			func(yield func(string, string) bool) {
				for _, pair := range [][2]string{{"Alice", "5"}, {"Bob", "3"}} {
					if !yield(pair[0], pair[1]) {
						return
					}
				}
			},
		)
	})
	if !strings.Contains(got, "Alice") || !strings.Contains(got, "Bob") {
		t.Errorf("KVTable: expected rows in output, got %q", got)
	}
	if !strings.Contains(got, "Name") || !strings.Contains(got, "Count") {
		t.Errorf("KVTable: expected headers in output, got %q", got)
	}
}

// --- Zero/empty inputs for all methods ---

func TestH1Empty(t *testing.T) {
	got := build(func(m *Builder) { m.H1("") })
	if got != "# \n\n" {
		t.Errorf("H1(empty): got %q", got)
	}
}

func TestParagraphEmpty(t *testing.T) {
	got := build(func(m *Builder) { m.Paragraph("") })
	if got != "\n\n" {
		t.Errorf("Paragraph(empty): got %q", got)
	}
}

func TestItalicParagraphEmpty(t *testing.T) {
	got := build(func(m *Builder) { m.ItalicParagraph("") })
	if got != "__\n\n" {
		t.Errorf("ItalicParagraph(empty): got %q", got)
	}
}

func TestKVEmpty(t *testing.T) {
	got := build(func(m *Builder) { m.KV("", "") })
	if got != "****: \n" {
		t.Errorf("KV(empty): got %q", got)
	}
}

func TestBulletListItemEmpty(t *testing.T) {
	// BulletListItem writes unconditionally; empty string produces "- \n".
	got := build(func(m *Builder) { m.BulletListItem("") })
	if got != "- \n" {
		t.Errorf("BulletListItem(empty): got %q", got)
	}
}

func TestBulletListAllEmpty(t *testing.T) {
	// BulletList filters zero values via ExtendBulletList/RemoveZeros.
	got := build(func(m *Builder) { m.BulletList("", "", "") })
	if got != "" {
		t.Errorf("BulletList(all empty): expected empty, got %q", got)
	}
}

func TestOrderedListItemEmpty(t *testing.T) {
	got := build(func(m *Builder) { m.OrderedListItem("") })
	if got != "1. \n" {
		t.Errorf("OrderedListItem(empty): got %q", got)
	}
}

func TestFencedCodeEmpty(t *testing.T) {
	// Empty code: WhenLine fires (len==0), producing a blank line between fences.
	got := build(func(m *Builder) { m.FencedCode("", "") })
	if got != "```\n\n```\n\n" {
		t.Errorf("FencedCode(empty,empty): got %q", got)
	}
}

func TestTextEmpty(t *testing.T) {
	got := build(func(m *Builder) { m.Text("") })
	if got != "" {
		t.Errorf("Text(empty): got %q", got)
	}
}

func TestBoldEmpty(t *testing.T) {
	got := build(func(m *Builder) { m.Bold("") })
	if got != "****" {
		t.Errorf("Bold(empty): got %q", got)
	}
}

func TestItalicEmpty(t *testing.T) {
	got := build(func(m *Builder) { m.Italic("") })
	if got != "__" {
		t.Errorf("Italic(empty): got %q", got)
	}
}

func TestPreformattedEmpty(t *testing.T) {
	got := build(func(m *Builder) { m.Preformatted("") })
	if got != "``" {
		t.Errorf("Preformatted(empty): got %q", got)
	}
}

func TestLinkEmpty(t *testing.T) {
	got := build(func(m *Builder) { m.Link("", "") })
	if got != "[]()" {
		t.Errorf("Link(empty): got %q", got)
	}
}

func TestStrikethroughEmpty(t *testing.T) {
	got := build(func(m *Builder) { m.Strikethrough("") })
	if got != "~~~~" {
		t.Errorf("Strikethrough(empty): got %q", got)
	}
}

func TestTableRowEmpty(t *testing.T) {
	// Row with no cells should not append a row; table stays empty and Build returns early.
	got := build(func(m *Builder) {
		m.NewTable(Column{Name: "X"}).Row().Build()
	})
	if got != "" {
		t.Errorf("Table.Row(no cells): expected empty output, got %q", got)
	}
}

func TestExtendRowEmpty(t *testing.T) {
	got := build(func(m *Builder) {
		m.NewTable(Column{Name: "X"}).
			ExtendRow(func(yield func(string) bool) {}).Build()
	})
	if got != "" {
		t.Errorf("Table.ExtendRow(empty seq): expected empty output, got %q", got)
	}
}

func TestKVTableEmpty(t *testing.T) {
	got := build(func(m *Builder) {
		m.KVTable(irt.MakeKV("K", "V"), func(yield func(string, string) bool) {})
	})
	if got != "" {
		t.Errorf("KVTable(empty seq): expected empty output, got %q", got)
	}
}

// --- runeByteOffset ---

func TestRuneByteOffset(t *testing.T) {
	b := []byte("F♯ Minor") // ♯ = 3 UTF-8 bytes; total 10 bytes, 8 runes
	cases := []struct{ n, want int }{
		{0, 0},
		{1, 1},  // after "F" (1 byte)
		{2, 4},  // after "F♯" (1+3 bytes)
		{8, 10}, // after all 8 runes = end of slice
		{99, 10}, // n > rune count → len(b)
	}
	for _, c := range cases {
		if got := runeByteOffset(b, c.n); got != c.want {
			t.Errorf("runeByteOffset(%q, %d) = %d, want %d", b, c.n, got, c.want)
		}
	}
}

// --- Unicode column widths ---

func TestTableUnicodeMusicalSymbols(t *testing.T) {
	// Regression: column widths were computed from byte length, causing
	// multi-byte Unicode characters (♯ = 3 UTF-8 bytes, 1 rune) to produce
	// under-padded cells.  Width must be rune count (visual width).
	//
	// "F♯ Minor" = 8 runes, 10 bytes.  Column width must be 8, not 10.
	got := build(func(m *Builder) {
		m.NewTable(Column{Name: "Key"}).
			Row("E Minor").  // 7 runes, 7 bytes
			Row("F♯ Minor"). // 8 runes, 10 bytes — ♯ is 3 UTF-8 bytes
			Build()
	})
	// width = max(runes("Key")=3, 7, 8) = 8
	want := "| Key      |\n| -------- |\n| E Minor  |\n| F♯ Minor |\n\n"
	if got != want {
		t.Errorf("TableUnicodeMusicalSymbols:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestTableUnicodeSmartQuotes(t *testing.T) {
	// Curly apostrophe ' (U+2019) is 3 UTF-8 bytes but 1 rune.
	got := build(func(m *Builder) {
		m.NewTable(Column{Name: "Title"}).
			Row("Short").          // 5 runes, 5 bytes
			Row("Saint\u2019s").   // 7 runes, 9 bytes
			Build()
	})
	// width = max(5, 5, 7) = 7
	want := "| Title   |\n| ------- |\n| Short   |\n| Saint\u2019s |\n\n"
	if got != want {
		t.Errorf("TableUnicodeSmartQuotes:\ngot:  %q\nwant: %q", got, want)
	}
}

func TestTableUnicodeColumnConsistency(t *testing.T) {
	// Every cell in a column must have the same visual width after padding.
	got := build(func(m *Builder) {
		m.NewTable(Column{Name: "K"}, Column{Name: "V"}).
			Row("plain", "A♭ Major"). // ♭ = 3 UTF-8 bytes
			Row("x", "B Major").
			Build()
	})
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) < 4 {
		t.Fatalf("expected at least 4 lines, got %d", len(lines))
	}
	// Collect visual widths of the second column cell across data rows.
	colWidths := make([]int, 0, len(lines)-2)
	for _, line := range lines[2:] {
		parts := strings.Split(line, "|")
		if len(parts) < 3 {
			continue
		}
		// parts[2] is " cell " — strip the two surrounding spaces.
		cell := parts[2]
		colWidths = append(colWidths, utf8.RuneCountInString(cell))
	}
	for i := 1; i < len(colWidths); i++ {
		if colWidths[i] != colWidths[0] {
			t.Errorf("column visual width inconsistent: row 0=%d row %d=%d\n%s",
				colWidths[0], i, colWidths[i], got)
		}
	}
}

// --- WriteTo ---

func TestWriteTo(t *testing.T) {
	var mb Builder
	mb.H1("Test")
	var out strings.Builder
	n, err := mb.WriteTo(&out)
	if err != nil {
		t.Fatal(err)
	}
	got := out.String()
	if got != "# Test\n\n" {
		t.Errorf("WriteTo: got %q", got)
	}
	if int(n) != len(got) {
		t.Errorf("WriteTo: reported n=%d but len=%d", n, len(got))
	}
}
