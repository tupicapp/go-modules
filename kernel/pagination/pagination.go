// Package pagination defines shared cursor-pagination types.
package pagination

type CursorPage struct {
	Cursor  *string
	PerPage int
}
