module github.com/sl1pm4t/k2tf

go 1.12

require (
	github.com/hashicorp/go-multierror v1.0.0
	github.com/hashicorp/hcl v1.0.0
	github.com/hashicorp/hcl2 v0.0.0-20190702185634-5b39d9ff3a9a
	github.com/hashicorp/terraform v0.12.5
	github.com/iancoleman/strcase v0.0.0-20180726023541-3605ed457bf7
	github.com/jinzhu/inflection v0.0.0-20180308033659-04140366298a
	github.com/mitchellh/reflectwalk v1.0.0
	github.com/rs/zerolog v1.11.0
	github.com/sirupsen/logrus v1.3.0
	github.com/spf13/pflag v1.0.3
	github.com/stretchr/testify v1.3.0
	github.com/terraform-providers/terraform-provider-kubernetes v1.10.0
	github.com/zclconf/go-cty v1.0.1-0.20190708163926-19588f92a98f
	k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/client-go v10.0.0+incompatible
	k8s.io/kube-aggregator v0.0.0-20190508191239-c5c2b08eec9f
)

replace git.apache.org/thrift.git => github.com/apache/thrift v0.12.0
