module github.com/sl1pm4t/k2tf

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
	github.com/terraform-providers/terraform-provider-kubernetes v1.9.0
	github.com/zclconf/go-cty v1.0.1-0.20190708163926-19588f92a98f
	k8s.io/api v0.0.0-20190222213804-5cb15d344471
	k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/client-go v10.0.0+incompatible
	k8s.io/kube-aggregator v0.0.0-20190508191239-c5c2b08eec9f
)

go 1.13

replace git.apache.org/thrift.git => github.com/apache/thrift v0.12.0

replace github.com/gophercloud/gophercloud v0.0.0-20190523203818-4885c347dcf4 => github.com/gophercloud/gophercloud v0.0.0-20190523203039-4885c347dcf4

replace github.com/keybase/go-crypto v0.0.0-20190523171820-b785b22cc757 => github.com/keybase/go-crypto v0.0.0-20190416182011-b785b22cc757

replace github.com/go-critic/go-critic v0.0.0-20181204210945-ee9bf5809ead => github.com/go-critic/go-critic v0.3.5-0.20190210220443-ee9bf5809ead

replace github.com/golangci/errcheck v0.0.0-20181003203344-ef45e06d44b6 => github.com/golangci/errcheck v0.0.0-20181223084120-ef45e06d44b6

replace github.com/golangci/go-tools v0.0.0-20180109140146-af6baa5dc196 => github.com/golangci/go-tools v0.0.0-20190318060251-af6baa5dc196

replace github.com/golangci/gofmt v0.0.0-20181105071733-0b8337e80d98 => github.com/golangci/gofmt v0.0.0-20181222123516-0b8337e80d98

replace github.com/golangci/gosec v0.0.0-20180901114220-66fb7fc33547 => github.com/golangci/gosec v0.0.0-20190211064107-66fb7fc33547

replace golang.org/x/tools v0.0.0-20190314010720-f0bfdbff1f9c => golang.org/x/tools v0.0.0-20190315214010-f0bfdbff1f9c

replace mvdan.cc/unparam v0.0.0-20190124213536-fbb59629db34 => mvdan.cc/unparam v0.0.0-20190209190245-fbb59629db34
