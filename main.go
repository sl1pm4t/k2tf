package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/hcl2/hclwrite"
	// "github.com/sl1pm4t/ky2tf/kubetf"
)

const y = `
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: baz-app
  namespace: bat
  annotations:
    foo: fam
spec:
  replicas: 2
  template:
    metadata:
      annotations:
        foo: fam
    spec:
      containers:
      - name: nginx
        image: nginx
        ports:
        - port: 80
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: bazApp
  namespace: bat
spec:
  replicas: 2
  template:
    metadata:
      annotations:
        foo: fam
    spec:
      containers:
      - name: nginx
        image: nginx
        ports:
        - port: 80
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: fooConfigMap
  namespace: bar
  labels:
    lbl1: somevalue
    lbl2: another
data:
  item1: wow
  item2: wee
`

func main() {
	spew.Config.Indent = "\t"
	r := strings.NewReader(y)
	objs, err := ParseK8SYAML(r)

	if err != nil {
		log.Fatal(err)
	}

	for _, obj := range objs {
		// kind := obj.GetObjectKind().GroupVersionKind().Kind
		// fmt.Println(kind)
		// fmt.Println(spew.Sdump(obj))

		// tfObj, err := kubetf.CreateTerraformResource(obj)

		// if err != nil {
		// 	log.Fatal(err)
		// }

		// WriteResources(os.Stdout, tfObj)

		f := hclwrite.NewEmptyFile()
		WriteObject(obj, f.Body())
		fmt.Fprint(os.Stdout, string(f.Bytes()))
	}

}

// func WriteResources(out io.Writer, res kubetf.Resource) error {
// 	hcl := &HCLFile{
// 		Resources: []kubetf.Resource{
// 			res,
// 		},
// 	}
// 	f := hclwrite.NewEmptyFile()
// 	gohcl.EncodeIntoBody(hcl, f.Body())
// 	// fmt.Printf("%s", f.Bytes())

// 	fmt.Fprint(out, string(f.Bytes()))
// 	return nil
// }

// type HCLFile struct {
// 	Datasources []kubetf.Resource `hcl:"datasource,block"`
// 	Resources   []kubetf.Resource `hcl:"resource,block"`
// }
