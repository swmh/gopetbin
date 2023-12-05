package main

import (
	"errors"
	"flag"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
)

type Config struct {
	targetPath string
	targetName string
	srcFile    string
	srcLine    int
	marshalers []Marshaler
}

func NewConfig() (*Config, error) {
	var config Config

	var isYaml bool
	flag.BoolVar(&isYaml, "yml", false, "yaml")
	flag.BoolVar(&isYaml, "yaml", false, "yaml")
	flag.BoolVar(&isYaml, "y", false, "yaml")

	var isEnv bool
	flag.BoolVar(&isEnv, "env", false, "env")
	flag.BoolVar(&isEnv, "e", false, "env")

	flag.StringVar(&config.targetPath, "path", "", "config path")
	flag.StringVar(&config.targetName, "name", "", "config name without extension")
	flag.Parse()

	if isYaml {
		config.marshalers = append(config.marshalers, NewYamlMarshaler(2))
	}

	if isEnv {
		config.marshalers = append(config.marshalers, &EnvMarshaler{})
	}

	config.srcFile = os.Getenv("GOFILE")
	if config.srcFile == "" {
		return nil, errors.New("GOFILE env must be set")
	}

	srcLine := os.Getenv("GOLINE")
	if srcLine == "" {
		return nil, errors.New("GOLINE env must be set")
	}

	var err error
	config.srcLine, err = strconv.Atoi(srcLine)

	if err != nil {
		log.Fatalf("cannot parse GOLINE: %s", err)
	}

	if config.targetName == "" {
		config.targetName = strings.Replace(config.srcFile, ".go", "", 1)
	}

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("cannot get working directory: %s\n", err)
	}

	config.srcFile = path.Join(cwd, config.srcFile)

	if config.targetPath == "" {
		config.targetPath = cwd
	}

	return &config, nil
}

func main() {
	config, err := NewConfig()
	if err != nil {
		log.Fatalf("Cannot initialize config: %s", err)
	}

	parser, err := NewParser(config.srcFile, config.srcLine)
	if err != nil {
		log.Fatalln(err)
	}

	if err := parser.Parse(); err != nil {
		log.Fatalf("Cannot parse: %s", err)
	}

	for _, v := range config.marshalers {
		file, err := os.Create(path.Join(config.targetPath, config.targetName))
		if err != nil {
			log.Fatalln(err)
		}
		defer file.Close()

		err = parser.Marshal(file, v)
		if err != nil {
			log.Fatalln(err)
		}
	}
}
