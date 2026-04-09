package mdwn

import (
	"bytes"
	"io"
	"iter"
	"strings"
	"unicode/utf8"

	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/strut"
)

const (
	newLineByte = '\n'
	newLineStr  = "\n"
)

// Column describes a single column in a markdown table.
type Column struct {
	Name        string
	RightAlign  bool
	MinWidth    int    // minimum column width; 0 means auto-size to content
	MaxWidth    int    // maximum column width; 0 means unlimited; truncates with TruncMarker
	TruncMarker string // suffix appended to truncated cells; defaults to "..."
}

func (c Column) truncMarker() string {
	if c.TruncMarker != "" {
		return c.TruncMarker
	}
	return "..."
}

// runeByteOffset returns the byte index of the start of the (n+1)th rune in b,
// i.e. the byte position immediately after n runes. Used to truncate cell
// content at a rune boundary rather than a byte boundary.
// Returns len(b) if b contains fewer than n runes.
func runeByteOffset(b []byte, n int) int {
	pos := 0
	for range n {
		if pos >= len(b) {
			return len(b)
		}
		_, size := utf8.DecodeRune(b[pos:])
		pos += size
	}

	return pos
}

// Builder wraps strut.Buffer with methods for writing markdown
// elements. All methods return the receiver for chaining. Call String() or
// WriteTo to get the result.
//
// strut.Buffer is used (over strut.Builder) because the primary output path
// is WriteTo — bytes.Buffer.WriteTo drains directly to the writer with no
// intermediate string copy, while strings.Builder.String() would require one.
// The trade-off is that String() itself copies; prefer WriteTo when possible.
type Builder struct{ strut.Mutable }

func MakeBuilder(capacity int) *Builder    { return &Builder{Mutable: *strut.MakeMutable(capacity)} }
func (m *Builder) Release()                { m.Mutable.Release() }
func (m *Builder) Truncate(targetSize int) { m.Mutable = m.Mutable[:max(0, min(targetSize, m.Len()))] }

// H1/H2/H3 write the heading followed by a blank line.
func (m *Builder) H1(text ...string) *Builder { return m.heading(1, text...) }
func (m *Builder) H2(text ...string) *Builder { return m.heading(2, text...) }
func (m *Builder) H3(text ...string) *Builder { return m.heading(3, text...) }

func (m *Builder) heading(level int, text ...string) *Builder {
	m.Concat(strings.Repeat("#", level), " ")
	m.Concat(text...)
	m.NLines(2)
	return m
}

// Paragraph writes text followed by a blank line.
func (m *Builder) Paragraph(text ...string) *Builder {
	m.Concat(text...)
	m.NLines(2)
	return m
}

// ItalicParagraph writes _text_ followed by a blank line.
func (m *Builder) ItalicParagraph(text string) *Builder {
	m.Italic(text)
	m.NLines(2)
	return m
}

// KV writes a **key**: value line followed by a newline.
func (m *Builder) KV(key, val string) *Builder {
	m.Concat("**", key, "**: ", val)
	m.Line()
	return m
}

// BulletListItem writes a single "- item" line.
func (m *Builder) BulletListItem(item string) *Builder {
	m.Concat("- ", item)
	m.Line()
	return m
}

// BulletList writes an unordered list followed by a blank line.
// Does nothing if no items are provided.
func (m *Builder) BulletList(items ...string) *Builder { return m.ExtendBulletList(irt.Slice(items)) }

// ExtendBulletList writes an unordered list from a sequence, followed by a
// blank line. Does nothing if the sequence is empty.
func (m *Builder) ExtendBulletList(seq iter.Seq[string]) *Builder {
	wrote := false
	for item := range irt.RemoveZeros(seq) {
		m.BulletListItem(item)
		wrote = true
	}
	m.WhenLine(wrote)
	return m
}

// OrderedListItem writes a single "1. item" line. Markdown renderers
// auto-number ordered list items regardless of the literal number used.
func (m *Builder) OrderedListItem(item string) *Builder {
	m.Concat("1. ", item)
	m.Line()
	return m
}

// OrderedList writes a numbered list followed by a blank line.
// Does nothing if no items are provided.
func (m *Builder) OrderedList(items ...string) *Builder {
	m.growToAccomidate(sumLengthOfStrings(items))
	for _, item := range items {
		m.OrderedListItem(item)
	}
	m.WhenLine(len(items) > 0)
	return m
}

func sumLengthOfStrings(strs []string) (total int) {
	for _, str := range strs {
		total += len(str)
	}
	return total
}

// growToAccomidate ensures at least l more bytes can be written without
// reallocation. bytes.Buffer.Grow(n) already accounts for existing free
// capacity, so passing l directly is correct and sufficient.
func (m *Builder) growToAccomidate(l int) { m.Grow(l) }

// ExtendOrderedList writes a numbered list from a sequence, followed by a
// blank line. Does nothing if the sequence is empty.
func (m *Builder) ExtendOrderedList(seq iter.Seq[string]) *Builder {
	var wrote bool
	for item := range irt.RemoveZeros(seq) {
		m.OrderedListItem(item)
		wrote = true
	}
	m.WhenLine(wrote)
	return m
}

// BlockQuote prefixes each line of text with "> " and appends a blank line.
// Blank lines within text are preserved so nested markdown elements render
// correctly. Trailing newlines are trimmed; a text consisting solely of
// newlines (or empty string) produces no output.
func (m *Builder) BlockQuote(text string) *Builder {
	// []byte(text) copies once; unavoidable with a string parameter.
	// Using bytes.TrimRight+SplitSeq directly avoids pool acquire/release
	// overhead — pools benefit same-sized reusable buffers, not one-off
	// allocations that vary with input length.
	b := bytes.TrimRight([]byte(text), newLineStr)
	if len(b) == 0 {
		return m
	}
	m.growToAccomidate(len(b))
	for line := range bytes.SplitSeq(b, []byte{newLineByte}) {
		m.PushString("> ")
		m.PushBytes(line)
		m.Line()
	}
	m.Line()
	return m
}

// BlockQuoteWith builds inner content using the provided function and wraps
// the result in a block quote, enabling nested block-quote elements.
// Uses the underlying byte buffer directly to avoid an intermediate string copy.
func (m *Builder) BlockQuoteWith(fn func(*Builder)) *Builder {
	var inner Builder
	fn(&inner)

	b := bytes.TrimRight(inner.Bytes(), newLineStr)
	if len(b) == 0 {
		return m
	}
	m.growToAccomidate(len(b))
	for line := range bytes.SplitSeq(b, []byte{newLineByte}) {
		m.PushString("> ")
		m.PushBytes(line)
		m.Line()
	}
	m.Line()
	return m
}

// FencedCode writes a fenced code block with an optional language identifier.
func (m *Builder) FencedCode(lang, code string) *Builder {
	m.Concat("```", lang)
	m.Line()
	m.PushString(code)
	m.WhenLine(len(code) == 0 || code[len(code)-1] != newLineByte)
	m.PushString("```")
	m.NLines(2)
	return m
}

// ParagraphBreak writes two newlines, creating a markdown paragraph separator.
func (m *Builder) ParagraphBreak() *Builder { m.NLines(2); return m }

// WriteTo drains the accumulated content to w without copying to an
// intermediate string.
func (m *Builder) WriteTo(w io.Writer) (int64, error) {
	n, err := w.Write(m.Mutable)
	return int64(n), err
}

// Text writes s to the builder and returns the receiver for chaining.
// Use this to intersperse plain text with inline formatting methods:
//
//	mb.Bold("Note").Text(": see ").Link("docs", url).ParagraphBreak()
func (m *Builder) Text(s string) *Builder { m.PushString(s); return m }

// Inline formatting methods — write directly to the builder and return the
// receiver for chaining.
// Example: mb.Bold("Note").Text(": see details.").ParagraphBreak()

func (m *Builder) Bold(s string) *Builder          { m.Concat("**", s, "**"); return m }
func (m *Builder) Italic(s string) *Builder        { m.Concat("_", s, "_"); return m }
func (m *Builder) Preformatted(s string) *Builder  { m.Concat("`", s, "`"); return m }
func (m *Builder) Link(text, url string) *Builder  { m.Concat("[", text, "](", url, ")"); return m }
func (m *Builder) Strikethrough(s string) *Builder { m.Concat("~~", s, "~~"); return m }

// NewTable creates a TableBuilder attached to this Builder. Call Row on the
// returned builder to accumulate rows, then Build to render the table and
// resume chaining on Builder.
func (m *Builder) NewTable(cols ...Column) *Table { return &Table{mb: m, cols: cols} }

// Table accumulates table rows and renders a column-aligned markdown
// table when Build is called. Cells are pipe-escaped at insertion time using
// pooled strut.Mutable buffers; Build releases them after rendering.
type Table struct {
	mb   *Builder
	cols []Column
	rows [][]*strut.Mutable
}

// ExtendRow appends a single data row from a sequence. Cells are
// pipe-escaped immediately. Iterates directly to avoid an intermediate
// []string allocation.
func (t *Table) ExtendRow(seq iter.Seq[string]) *Table {
	var row []*strut.Mutable
	for cell := range seq {
		m := strut.NewMutable()
		m.WithReplaceAll(cell, "|", `\|`)
		row = append(row, m)
	}
	if len(row) > 0 {
		t.rows = append(t.rows, row)
	}
	return t
}

// Row appends a single data row. Cells are pipe-escaped immediately.
func (t *Table) Row(cells ...string) *Table {
	if len(cells) == 0 {
		return t
	}
	// make avoids the irt.GenerateN iterator + irt.Collect overhead.
	row := make([]*strut.Mutable, len(cells))
	for i, cell := range cells {
		row[i] = strut.NewMutable()
		row[i].WithReplaceAll(cell, "|", `\|`)
	}
	t.rows = append(t.rows, row)
	return t
}

// Rows appends multiple data rows to the table.
func (t *Table) Rows(rows [][]string) *Table { return t.Extend(irt.Slice(rows)) }

// Extend appends rows from a sequence to the table.
func (t *Table) Extend(seq iter.Seq[[]string]) *Table {
	for row := range seq {
		t.Row(row...)
	}
	return t
}

// Build renders the accumulated table into the parent Builder and returns it
// for further chaining. Column widths are auto-sized to content,
// lower-bounded by ColumnDef.MinWidth, at least 3 (minimum for a valid
// markdown separator), and capped by ColumnDef.MaxWidth when set.
// Cells exceeding MaxWidth are truncated with ColumnDef.TruncMarker ("...").
// The pooled Mutable cell buffers are released after rendering.
func (t *Table) Build() *Builder {
	if len(t.rows) == 0 {
		return t.mb
	}
	defer func() {
		for i := range t.rows {
			for j := range t.rows[i] {
				t.rows[i][j].Release()
				t.rows[i][j] = nil
			}
			t.rows[i] = nil
		}
		t.rows = nil
	}()

	// Compute per-column widths using rune count (visual width), not byte
	// length. Multi-byte Unicode characters (e.g. ♯ = 3 UTF-8 bytes, 1 rune)
	// must count as one column character to keep rows visually aligned.
	widths := make([]int, len(t.cols))
	for i, col := range t.cols {
		widths[i] = max(utf8.RuneCountInString(col.Name), col.MinWidth, 3)
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if i < len(widths) {
				if rw := utf8.RuneCount([]byte(*cell)); rw > widths[i] {
					widths[i] = rw
				}
			}
		}
	}
	// Apply MaxWidth cap.
	for i, col := range t.cols {
		if col.MaxWidth > 0 && widths[i] > col.MaxWidth {
			widths[i] = max(col.MaxWidth, col.MinWidth, 3)
		}
	}

	// Pre-grow the output buffer: each row is ~(sum of widths + 3 per col + newline).
	rowWidth := 1 // leading "|"
	for _, w := range widths {
		rowWidth += w + 3 // " cell |"
	}
	t.mb.Grow(rowWidth * (len(t.rows) + 2)) // +2 for header and separator rows

	// Header row.
	t.mb.PushString("|")
	for i, col := range t.cols {
		t.mb.Concat(" ", col.Name, strings.Repeat(" ", widths[i]-utf8.RuneCountInString(col.Name)))
		t.mb.PushString(" |")
	}
	t.mb.Line()

	// Separator row: right-aligned columns use "----:" syntax.
	t.mb.PushString("|")
	for i, col := range t.cols {
		t.mb.PushString(" ")
		if col.RightAlign {
			t.mb.Concat(strings.Repeat("-", widths[i]-1), ":")
		} else {
			t.mb.PushString(strings.Repeat("-", widths[i]))
		}
		t.mb.PushString(" |")
	}
	t.mb.Line()

	// Data rows.
	for _, row := range t.rows {
		t.mb.PushString("|")
		for i, col := range t.cols {
			// Get the raw escaped bytes for this cell (nil = empty).
			var cellBytes []byte
			if i < len(row) && row[i] != nil {
				cellBytes = []byte(*row[i])
			}
			cellLen := utf8.RuneCount(cellBytes)

			// Truncate if cell exceeds capped column width.
			needsTrunc := col.MaxWidth > 0 && cellLen > widths[i]
			if needsTrunc {
				marker := col.truncMarker()
				markerLen := utf8.RuneCountInString(marker)
				if widths[i] > markerLen {
					cutAt := runeByteOffset(cellBytes, widths[i]-markerLen)
					cellBytes = append(cellBytes[:cutAt:cutAt], marker...)
				} else {
					cellBytes = cellBytes[:runeByteOffset(cellBytes, widths[i])]
				}
				cellLen = widths[i]
			}

			pad := widths[i] - cellLen
			t.mb.PushString(" ")
			if col.RightAlign && pad > 0 {
				t.mb.PushString(strings.Repeat(" ", pad))
			}
			t.mb.PushBytes(cellBytes)
			if !col.RightAlign && pad > 0 {
				t.mb.PushString(strings.Repeat(" ", pad))
			}
			t.mb.PushString(" |")
		}
		t.mb.Line()
	}

	return t.mb
}

// KVTable builds a two-column key/value table from a two-value sequence and
// calls Build. The header parameter names the key and value columns. Values
// are formatted with fmt.Sprint.
func (mb *Builder) KVTable(header irt.KV[string, string], seq iter.Seq2[string, string]) *Builder {
	tb := mb.NewTable(Column{Name: header.Key}, Column{Name: header.Value, RightAlign: true})
	irt.Apply2(seq, tb.kvRow)
	return tb.Build()
}

func (tb *Table) kvRow(k, v string) { tb.Row(k, v) }
