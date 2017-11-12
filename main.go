package main

import (
	"fmt"
	"io"
	"log"
	"os"

	"gopkg.in/docopt/docopt.go.v0"
	"gopkg.in/hashicorp/logutils.v0"
)

//go:generate go run misc/include_jsons.go

const (
	pkgVersion = "0.5.0"
	pkgTitle   = "testman/" + pkgVersion
	envAPIKey  = "COVEREDCI_API_KEY"
)

var (
	isDebug bool
	apiRoot string
	initURL string
	nextURL string
	docsURL string
)

func init() {
	log.SetFlags(log.Lshortfile | log.Lmicroseconds | log.LUTC)

	isDebug = "1" == os.Getenv("DEBUG")

	if isDebug {
		apiRoot = "http://localhost:1042/1"
		docsURL = "http://localhost:2042/1/blob"
	} else {
		apiRoot = "https://test.coveredci.com/1"
		docsURL = "https://lint.coveredci.com/1/blob"
	}
	initURL = apiRoot + "/init"
	nextURL = apiRoot + "/next"

	unstacheInit()
}

func main() {
	os.Exit(actualMain())
}

func usage() (map[string]interface{}, error) {
	usage := `testman

Usage:
  testman [-vvv] test
  testman [-vvv] validate
  testman -h | --help
  testman -V | --version

Options:
  -v, -vv, -vvv  Verbosity level
  -h, --help     Show this screen
  -V, --version  Show version`

	return docopt.Parse(usage, nil, true, pkgTitle, true)
}

func actualMain() int {
	args, err := usage()
	if err != nil {
		log.Println("!args: ", err)
		return 1
	}

	logCatchall, err := os.OpenFile("/tmp/testman.log", os.O_WRONLY|os.O_CREATE, 0640)
	if err != nil {
		log.Println(err)
		return 1
	}
	defer logCatchall.Close()
	logFiltered := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DBG", "WRN", "ERR", "NOP"},
		MinLevel: logLevel(args),
		Writer:   os.Stderr,
	}
	log.SetOutput(io.MultiWriter(logCatchall, logFiltered))
	log.Println("[ERR]", pkgTitle, args)

	if !isDebug {
		latest := getLatestRelease()
		if isOutOfDate(pkgVersion, latest) {
			log.Printf("A newer version of %s is available: %s\n", pkgTitle, latest)
			return 3
		}
	}

	if _, err := os.Stat(shell()); os.IsNotExist(err) {
		log.Println(shell() + " is required")
		return 5
	}

	apiKey := getAPIKey()
	if args["validate"].(bool) {
		yml := readYAML(localYML)
		_, errors := validateDocs(apiKey, yml)
		if errors != nil {
			reportValidationErrors(errors)
			return 2
		} else {
			fmt.Println("No validation errors found.")
			//TODO: make it easy to use returned token
			return 0
		}
	}

	if apiKey == "" {
		log.Println("$" + envAPIKey + " is unset")
		return 4
	}

	envSerializedPath := uniquePath()
	ensureDeleted(envSerializedPath)
	snapEnv(envSerializedPath)
	defer ensureDeleted(envSerializedPath)

	cfg, cmd := initDialogue(apiKey)
	log.Printf("[DBG] init cmd: %+v\n", cmd)
	for {
		cmd = next(cfg, cmd)
		if nil == cmd {
			log.Println("We're done!")
			return 0
		}
	}
}

func ensureDeleted(path string) {
	if err := os.Remove(path); err != nil && os.IsExist(err) {
		log.Fatal("[ERR] ", err)
	}
}

func getAPIKey() string {
	apiKey := os.Getenv(envAPIKey)
	if isDebug {
		apiKey = "42"
	}
	return apiKey
}

func logLevel(args map[string]interface{}) logutils.LogLevel {
	var lvl string
	switch args["-v"].(int) {
	case 1:
		lvl = "ERR"
	case 2:
		lvl = "WRN"
	case 3:
		lvl = "DBG"
	default:
		lvl = "NOP"
	}
	return logutils.LogLevel(lvl)
}
