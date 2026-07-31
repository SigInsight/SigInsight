package litequery

type Statement struct {
	Name     string
	SQL      string
	Args     []any
	Columns  []ResultColumn
	Warnings []string
}

// ResultColumn preserves semantic result names without using user-controlled
// field text as a SQL identifier or alias.
type ResultColumn struct {
	Name  string
	Field *FieldRef
}
