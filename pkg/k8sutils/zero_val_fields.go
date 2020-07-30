package k8sutils

var includedZeroValFields = []string{
	"RunAsUser", // container.securityContext.runAsUser -- 0 means run as root
}
var includeZeroValFieldMap map[string]bool

func init() {
	includeZeroValFieldMap = make(map[string]bool, len(includedZeroValFields))
	for _, v := range includedZeroValFields {
		includeZeroValFieldMap[v] = true
	}
}

// IncludedZeroValField checks a list of FieldNames to determine if they should be
// included in HCL output even though the golang variable is Zero and would
// normally be assumed to be unset
func IncludedZeroValField(name string) bool {
	_, ok := includeZeroValFieldMap[name]
	return ok
}
