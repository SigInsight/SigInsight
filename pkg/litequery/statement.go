package litequery

type Statement struct {
	Name        string
	SQL         string
	Args        []any
	Columns     []ResultColumn
	Warnings    []string
	Pagination  *Pagination
	ResultLimit uint32
}

// Pagination describes a normal result page. The compiler requests one extra
// row so the executor can report whether another page exists without counting
// the full result set.
type Pagination struct {
	Limit  uint32
	Offset uint32
}

// ResultColumn preserves semantic result names without using user-controlled
// field text as a SQL identifier or alias.
type ResultColumn struct {
	Name  string
	Field *FieldRef
}
