package msgui

import (
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
	default:
		yield(md, nil)
	}
}
