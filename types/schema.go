package types

type ColumnSchema struct {
	Column        string            `json:"column"`
	TargetColumn  string            `json:"target_column,omitempty"`
	Values        []string          `json:"values"`
	ValuesMapping map[string]string `json:"values_mapping,omitempty"`
	// TokenMapping maps the parts of grouped values (R+W+D) so combinations
	// not seen in the sample still convert, joined with TokenSeparator
	TokenMapping   map[string]string `json:"token_mapping,omitempty"`
	TokenSeparator string            `json:"token_separator,omitempty"`
	// Unresolved lists source values with no mapping, for manual review
	Unresolved []string `json:"unresolved,omitempty"`
	// Evidence explains how the mapping was found
	Evidence string `json:"evidence,omitempty"`
}
