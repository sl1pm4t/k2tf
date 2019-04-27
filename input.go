package main

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func readInput() []runtime.Object {
	if input == "-" || input == "" {
		return readStdinInput()
	}
	return readFilesInput()
}

func readStdinInput() []runtime.Object {
	var objs []runtime.Object

	info, err := os.Stdin.Stat()
	if err != nil {
		panic(err)
	}

	if info.Mode()&os.ModeCharDevice != 0 {
		log.Fatal().Msg("No data read from stdin")
	}

	reader := bufio.NewReader(os.Stdin)
	parsed, err := parseK8SYAML(reader)
	if err != nil {
		log.Fatal().Err(err)
	}

	for _, obj := range parsed {
		if obj.GetObjectKind().GroupVersionKind().Kind == "List" {
			list := obj.(*corev1.List)
			for _, item := range list.Items {
				itemObj, err := parseK8SJSON(item.Raw)
				if err != nil {
					log.Error().Err(err)
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

	if _, err := os.Stat(input); os.IsNotExist(err) {
		log.Fatal().Str("file", input).Msg("input filepath does not exist")
	}

	file, err := os.Open(input)
	if err != nil {
		log.Fatal().Err(err)
	}

	fs, err := file.Stat()
	if err != nil {
		log.Fatal().Err(err)
	}

	readFile := func(fileName string) {
		log.Debug().Msgf("reading file: %s", fileName)
		content, err := ioutil.ReadFile(fileName)
		if err != nil {
			log.Fatal().Err(err)
		}

		r := bytes.NewReader(content)
		obj, err := parseK8SYAML(r)
		if err != nil {
			log.Fatal().Err(err)
		}
		objs = append(objs, obj...)
	}

	if fs.Mode().IsDir() {
		// read directory
		log.Debug().Msgf("reading directory: %s", input)

		dirContents, err := file.Readdirnames(0)
		if err != nil {
			log.Fatal().Err(err)
		}

		for _, f := range dirContents {
			if strings.HasSuffix(f, ".yml") || strings.HasSuffix(f, ".yaml") {
				readFile(filepath.Join(input, f))
			}
		}

	} else {
		// read single file
		readFile(input)

	}

	return objs
}
