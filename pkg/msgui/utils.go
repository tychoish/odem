package msgui

import (
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/mdwn"
)

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
