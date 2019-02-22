package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/hcl2/hclwrite"
)

var debug bool = true

func main() {
	spew.Config.Indent = "\t"
	r := strings.NewReader(yd1)
	objs, err := ParseK8SYAML(r)

	if err != nil {
		log.Fatal(err)
	}

	for _, obj := range objs {
		f := hclwrite.NewEmptyFile()
		WriteObject(obj, f.Body())
		fmt.Fprint(os.Stdout, string(f.Bytes()))
	}

}
