package domain

// Property kinds.
var PropertyKinds = []string{"single_family", "multi_family", "condo", "townhouse", "commercial", "other"}

// OneOf reports whether v is in options.
func OneOf(v string, options []string) bool {
	for _, o := range options {
		if o == v {
			return true
		}
	}
	return false
}
