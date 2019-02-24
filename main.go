package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/hashicorp/hcl2/hclwrite"
)

var debug bool

func init() {
	flag.BoolVar(&debug, "debug", false, "enable debug mode")

	spew.Config.Indent = "\t"
}

func main() {
	flag.Parse()

	r := strings.NewReader(podVolumesOnlyYAML)
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
