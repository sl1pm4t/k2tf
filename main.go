package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl2/hclwrite"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"

	corev1 "k8s.io/api/core/v1"
)

// Command line flags
var (
	debug bool
	input string
	// output string
)

func init() {
	// init command line flags
	flag.BoolVar(&debug, "debug", false, "enable debug mode")
	const inputUsage = `file or directory that contains the YAML configuration to convert. Use "-" to read from stdin.`
	flag.StringVar(&input, "filepath", "-", inputUsage)
	flag.StringVar(&input, "f", "-", inputUsage+" (shorthand)")
	// const outputUsage = `file or directory where Terraform config will be written`
	// flag.StringVar(&output, "output", "-", outputUsage)
	// flag.StringVar(&output, "o", "-", outputUsage+" (shorthand)")

	log.SetOutput(os.Stderr)
}

func main() {
	flag.Parse()

	objs := readInput()

	for _, obj := range objs {
		f := hclwrite.NewEmptyFile()
		WriteObject(obj, f.Body())
		fmt.Fprint(os.Stdout, string(f.Bytes()))
	}

}

func readInput() []runtime.Object {
	if input == "-" || input == "" {
		return readStdinInput()
	} else {
		return readFilesInput()
	}
}

func readStdinInput() []runtime.Object {
	var objs []runtime.Object

	info, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}

	if info.Mode()&os.ModeCharDevice != 0 { //|| info.Size() <= 0 {
		log.Fatalf("No data read from stdin")
	}

	reader := bufio.NewReader(os.Stdin)
	parsed, err := ParseK8SYAML(reader)
	if err != nil {
		log.Fatal(err)
	}

	for _, obj := range parsed {
		log.Infof("%T", obj)
		log.Infof("%s", obj.GetObjectKind())
		if obj.GetObjectKind().GroupVersionKind().Kind == "List" {
			list := obj.(*corev1.List)
			for _, item := range list.Items {
				itemObj, err := ParseK8SJSON(item.Raw)
				if err != nil {
					log.Error(err)
					continue
				}
				objs = append(objs, itemObj)

			}

		} else {
			objs = append(objs, obj)

		}
	}

	return objs
}

func readFilesInput() []runtime.Object {
	var objs []runtime.Object

	file, err := os.Open(input) // For read access.
	if err != nil {
		log.Fatal(err)
	}

	fs, err := file.Stat()
	if err != nil {
		log.Fatal(err)
	}

	readFile := func(fileName string) {
		content, err := ioutil.ReadFile(fileName)
		if err != nil {
			log.Fatal(err)
		}

		r := bytes.NewReader(content)
		obj, err := ParseK8SYAML(r)
		if err != nil {
			log.Fatal(err)
		}
		objs = append(objs, obj...)
	}

	if fs.Mode().IsDir() {
		// read directory
		dirContents, err := file.Readdirnames(0)
		if err != nil {
			log.Fatal(err)
		}

		for _, f := range dirContents {
			if strings.HasSuffix(f, ".yml") || strings.HasSuffix(f, ".yaml") {
				readFile(filepath.Join(fs.Name(), f))
			}
		}

	} else {
		// read single file
		readFile(input)

	}

	return objs
}
