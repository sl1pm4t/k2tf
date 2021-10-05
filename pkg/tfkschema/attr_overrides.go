package tfkschema

// IncludedOnZero checks the attribute name against a lookup table to determine if it can be included
// when it is zero / empty.
func IncludedOnZero(attrName string) bool {
	switch attrName {
	case "EmptyDir":
		return true
	case "RunAsUser":
		return true
	}
	return false
}