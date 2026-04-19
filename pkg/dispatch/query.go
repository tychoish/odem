package dispatch

import (
	"fmt"

	"github.com/tychoish/fun/ers"
)

type MinutesAppQueryType int

const (
	MinutesAppQueryTypeUnknown MinutesAppQueryType = iota
	MinutesAppQueryTypeLeader
	MinutesAppQueryTypeSong
	MinutesAppQueryTypeSinging
	MinutesAppQueryTypeLocality
	MinutesAppQueryTypeYear
	MinutesAppQueryTypeKey
	MinutesAppQueryTypeOperation
	MinutesAppQueryTypeWord
	// MinutesAppQueryTypeDocumentOutput signals that the result should be
	// rendered as a file attachment rather than chat messages. It requires no
	// user input and is auto-satisfied by discoverNext.
	MinutesAppQueryTypeDocumentOutput
	MinutesAppQueryTypeInvalid
)

func (maqt MinutesAppQueryType) Ok() bool { return maqt > 0 && maqt < MinutesAppQueryTypeInvalid }
func (maqt MinutesAppQueryType) Validate() error {
	switch {
	case maqt.Ok():
		return nil
	case maqt == MinutesAppQueryTypeUnknown:
		return ers.Error("undefined query")
	case maqt == MinutesAppQueryTypeInvalid:
		return ers.Error("invalid query")
	default:
		return ers.Error("undefined invalid")
	}
}

func (maqt MinutesAppQueryType) String() string {
	switch maqt {
	case MinutesAppQueryTypeUnknown:
		return "unknown"
	case MinutesAppQueryTypeLeader:
		return "leader"
	case MinutesAppQueryTypeSong:
		return "song"
	case MinutesAppQueryTypeSinging:
		return "singing"
	case MinutesAppQueryTypeLocality:
		return "locality"
	case MinutesAppQueryTypeYear:
		return "year"
	case MinutesAppQueryTypeKey:
		return "key"
	case MinutesAppQueryTypeOperation:
		return "operation"
	case MinutesAppQueryTypeWord:
		return "word"
	case MinutesAppQueryTypeDocumentOutput:
		return "document-output"
	case MinutesAppQueryTypeInvalid:
		return "invalid"
	default:
		return fmt.Sprintf("MinutesAppQueryType<%d>[undefined]", maqt)
	}
}
