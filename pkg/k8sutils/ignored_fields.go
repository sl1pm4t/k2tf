package k8sutils

var ignoredFields = []string{
	"CreationTimestamp",
	"DeletionTimestamp",
	"Generation",
	"OwnerReferences",
	"ResourceVersion",
	"SelfLink",
	"TypeMeta",
	"Status",
	"UID",
}
var ignoredFieldMap map[string]bool

func init() {
	ignoredFieldMap = make(map[string]bool, len(ignoredFields))
	for _, v := range ignoredFields {
		ignoredFieldMap[v] = true
	}
}

func IgnoredField(name string) bool {
	_, ok := ignoredFieldMap[name]
	return ok
}
