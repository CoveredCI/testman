package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/go-yaml/yaml"
)

const (
	localYML       = ".fuzzymonkey.yml"
	lastYMLVersion = 1
	defaultYMLHost = "localhost"
	defaultYMLPort = "3000"
)

type ymlCfg struct {
	AuthToken string
	File      string
	Kind      string
	Host      string
	Port      string
	FinalHost string
	FinalPort string
	Start     []string
	Reset     []string
	Stop      []string
}

func newCfg(yml []byte, showCfg bool) (cfg *ymlCfg, err error) {
	var vsn struct {
		V interface{} `yaml:"version"`
	}
	if vsnErr := yaml.Unmarshal(yml, &vsn); vsnErr != nil {
		err = fmt.Errorf("field 'version' missing! Try `version: %d`",
			lastYMLVersion)
		log.Println("[ERR]", err)
		colorERR.Println(err)
		return
	}

	version, ok := vsn.V.(int)
	if !ok || !knownVersion(version) {
		err = fmt.Errorf("bad version: `%+v'", vsn.V)
		log.Println("[ERR]", err)
		colorERR.Println(err)
		return
	}

	type cfgParser func(yml []byte, showCfg bool) (cfg *ymlCfg, err error)
	cfgParsers := []cfgParser{
		newCfgV001,
	}

	return cfgParsers[version-1](yml, showCfg)
}

func knownVersion(v int) bool {
	if 0 < v && v <= lastYMLVersion {
		return true
	}
	return false
}

func newCfgV001(yml []byte, showCfg bool) (cfg *ymlCfg, err error) {
	var ymlConf struct {
		V     uint     `yaml:"version"`
		Start []string `yaml:"start"`
		Reset []string `yaml:"reset"`
		Stop  []string `yaml:"stop"`
		Doc   struct {
			File string `yaml:"file"`
			Kind string `yaml:"kind"`
			Host string `yaml:"host"`
			Port string `yaml:"port"`
		} `yaml:"documentation"`
	}

	if err = yaml.UnmarshalStrict(yml, &ymlConf); err != nil {
		log.Println("[ERR]", err)
		colorERR.Println("Failed to parse", localYML)
		r := strings.NewReplacer("not found", "unknown")
		for _, e := range strings.Split(err.Error(), "\n") {
			if end := strings.Index(e, " in type struct"); end != -1 {
				colorERR.Println(r.Replace(e[:end]))
			}
		}
		return
	}

	if ymlConf.Doc.Host == "" {
		def := defaultYMLHost
		log.Printf("[NFO] field 'host' is empty/unset: using %v\n", def)
		ymlConf.Doc.Host = def
	}

	if ymlConf.Doc.Port == "" {
		def := defaultYMLPort
		log.Printf("[NFO] field 'port' is empty/unset: using %v\n", def)
		ymlConf.Doc.Port = def
	}

	if showCfg {
		enc := yaml.NewEncoder(os.Stderr)
		defer enc.Close()
		if err = enc.Encode(ymlConf); err != nil {
			log.Println("[ERR]", err)
			colorERR.Printf("Failed to pretty-print %s: %+v\n", localYML, err)
			return
		}
	}

	cfg = &ymlCfg{
		File:  ymlConf.Doc.File,
		Kind:  ymlConf.Doc.Kind,
		Host:  ymlConf.Doc.Host,
		Port:  ymlConf.Doc.Port,
		Start: ymlConf.Start,
		Reset: ymlConf.Reset,
		Stop:  ymlConf.Stop,
	}

	return
}

func (cfg *ymlCfg) script(kind cmdKind) []string {
	return map[cmdKind][]string{
		kindStart: cfg.Start,
		kindReset: cfg.Reset,
		kindStop:  cfg.Stop,
	}[kind]
}

func (cfg *ymlCfg) findBlobs() (path string, err error) {
	//FIXME: force relative paths & nested under workdir. Watch out for links
	path = cfg.File
	if len(path) == 0 {
		err = fmt.Errorf("Path to documentation is empty")
		log.Println("[ERR]", err)
		colorERR.Println(err)
		return
	}

	log.Println("[NFO] spec is at", path)
	return
}

func readYML() (yml []byte, err error) {
	fd, err := os.Open(localYML)
	if err != nil {
		log.Println("[ERR]", err)
		colorERR.Printf("You must provide a readable %s file in the current directory.\n", localYML)
		return
	}
	defer fd.Close()

	if yml, err = ioutil.ReadAll(fd); err != nil {
		log.Println("[ERR]", err)
	}
	return
}
