package store

import (
	"strconv"
	"strings"
)

func itoa(i int) string { return strconv.Itoa(i) }

// filter builds a WHERE clause from optional conditions. Conditions use ? for
// their value, which becomes the next numbered placeholder.
type filter struct {
	conds []string
	args  []any
}

// newFilter starts a filter whose first arguments are fixed, usually the
// owner ID as $1.
func newFilter(args ...any) *filter {
	return &filter{args: args}
}

// add appends a condition with one value.
func (f *filter) add(cond string, v any) {
	f.args = append(f.args, v)
	f.conds = append(f.conds, strings.ReplaceAll(cond, "?", "$"+itoa(len(f.args))))
}

// raw appends a condition that takes no value.
func (f *filter) raw(cond string) {
	f.conds = append(f.conds, cond)
}

// arg adds a value without a condition and returns its placeholder, for
// LIMIT and OFFSET.
func (f *filter) arg(v any) string {
	f.args = append(f.args, v)
	return "$" + itoa(len(f.args))
}

// where returns " WHERE ..." or "" when there are no conditions.
func (f *filter) where() string {
	if len(f.conds) == 0 {
		return ""
	}
	return " WHERE " + strings.Join(f.conds, " AND ")
}
