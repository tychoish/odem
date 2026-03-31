package models

type Params struct {
	// Provide input for the name of the singer, the song or the
	// singing, as relevant to the query.
	Name string

	// Years makes it possible to limit the scope of a query to
	// specific years. Negative numbers exclude years from
	// queries, positive numbers include them. When empty query
	// all years. This is always optional.
	Years []int

	// Limit the number of items returned to this number:
	// typically the handlers will restrict this to somewhere
	// between 16 and 64 depending, but it can be overridden.
	Limit int
}
