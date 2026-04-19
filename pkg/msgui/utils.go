package msgui

import (
	"fmt"
	"iter"

	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/mdwn"
)

func renderLineItems[T interface{ LineItem() *mdwn.Builder }](records iter.Seq[T]) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		mdtb := mdwn.MakeBuilder(4096)
		for record := range records {
			line := record.LineItem()
			if mdtb.Len() >= 4000 || mdtb.Len()+line.Len() >= 4000 {
				if !yield(mdtb, nil) {
					return
				}
				mdtb = mdwn.MakeBuilder(4096)
			}
			(&mdtb.Mutable).WriteMutableLine(line.Mutable)
		}
		flush(mdtb, yield)
	}
}

// renderWithHeader combines the header with result chunks so the first
// message contains the header followed by a blank line and the first page of
// results. Continuation messages prepend "header (cont'd; N/)" so the context
// is clear even without the initial message.
func renderWithHeader[T interface{ LineItem() *mdwn.Builder }](header *mdwn.Builder, records iter.Seq[T]) iter.Seq2[*mdwn.Builder, error] {
	return func(yield func(*mdwn.Builder, error) bool) {
		headerText := header.String()
		contNum := 0

		newBuilder := func() *mdwn.Builder {
			mdtb := mdwn.MakeBuilder(4096)
			if contNum == 0 {
				mdtb.PushString(headerText)
			} else {
				mdtb.Concat(headerText, fmt.Sprintf(" (cont'd; %d/)", contNum))
			}
			mdtb.PushString("\n\n")
			return mdtb
		}

		mdtb := newBuilder()
		added := false

		for record := range records {
			added = true
			line := record.LineItem()
			if mdtb.Len() >= 4000 || mdtb.Len()+line.Len() >= 4000 {
				if !yield(mdtb, nil) {
					return
				}
				contNum++
				mdtb = newBuilder()
			}
			(&mdtb.Mutable).WriteMutableLine(line.Mutable)
		}

		if !added {
			yield(nil, ers.New("empty results"))
			return
		}
		flush(mdtb, yield)
	}
}

func flush(md *mdwn.Builder, yield func(*mdwn.Builder, error) bool) {
	switch {
	case md.Len() > 4096:
		errb := mdwn.MakeBuilder(256)
		errb.Bold("ERROR:").Mprintf("response output (%d) exceeded max size 4096.", md.Len())
		md.Truncate(4095)
		if !yield(errb, nil) || !yield(md, nil) {
			return
		}
		yield(nil, ers.New("oversize abort builder"))
	case md.Len() <= 4:
		yield(nil, ers.New("empty results"))
	default:
		yield(md, nil)
	}
}
