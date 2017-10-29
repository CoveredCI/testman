package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"time"
)

type reqCmd struct {
	V       uint        `json:"v"`
	Cmd     string      `json:"cmd"`
	UID     interface{} `json:"uid"`
	Method  string      `json:"method"`
	Url     string      `json:"url"`
	Headers []string    `json:"headers"`
	Payload *string     `json:"payload"`
}

type reqCmdRepOK struct {
	Cmd     string      `json:"cmd"`
	V       uint        `json:"v"`
	Us      uint64      `json:"us"`
	UID     interface{} `json:"uid"`
	Code    int         `json:"code"`
	Headers []string    `json:"headers"`
	Payload string      `json:"payload"`
}

type reqCmdRepKO struct {
	Cmd    string      `json:"cmd"`
	V      uint        `json:"v"`
	Us     uint64      `json:"us"`
	UID    interface{} `json:"uid"`
	Reason string      `json:"reason"`
}

func (cmd reqCmd) Kind() string {
	return cmd.Cmd
}

func (cmd reqCmd) Exec(_cfg *ymlCfg) []byte {
	ok, ko := makeRequest(cmd)

	var rep []byte
	var err error
	if ok != nil {
		rep, err = json.Marshal(ok)
	} else {
		rep, err = json.Marshal(ko)
	}

	if err != nil {
		log.Fatal("!encode: ", err)
	}
	return rep
}

func makeRequest(cmd reqCmd) (*reqCmdRepOK, *reqCmdRepKO) {
	var r *http.Request
	var err error
	if cmd.Payload != nil {
		inPayload := bytes.NewBufferString(*cmd.Payload)
		r, err = http.NewRequest(cmd.Method, cmd.Url, inPayload)
	} else {
		r, err = http.NewRequest(cmd.Method, cmd.Url, nil)
	}
	if err != nil {
		log.Fatal("!NewRequest: ", err)
	}

	for _, header := range cmd.Headers {
		if header == "User-Agent: CoveredCI-passthrough/1" {
			r.Header.Set("User-Agent", pkgVersion)
		} else {
			pair := strings.SplitN(header, ": ", 2)
			r.Header.Set(pair[0], pair[1])
		}
	}
	client := &http.Client{}
	start := time.Now()
	resp, err := client.Do(r)
	us := uint64(time.Since(start) / time.Microsecond)
	var _pld string
	if nil == cmd.Payload {
		_pld = ""
	} else {
		_pld = *cmd.Payload
	}

	if err != nil {
		reason := fmt.Sprintf("%+v", err.Error())
		log.Printf("🡳  %vμs %s %s\n  ▲  %s\n  ▼  %s\n", us, cmd.Method, cmd.Url, _pld, reason)
		ko := &reqCmdRepKO{
			V:      1,
			Cmd:    cmd.Cmd,
			UID:    cmd.UID,
			Us:     us,
			Reason: reason,
		}
		return nil, ko

	} else {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatal("!read body: ", err)
		}
		log.Printf("🡳  %vμs %s %s\n  ▲  %s\n  ▼  %s\n", us, cmd.Method, cmd.Url, _pld, body)
		var headers []string
		//// headers = append(headers, fmt.Sprintf("Host: %v", resp.Host))
		// Loop through headers
		//FIXME: preserve order github.com/golang/go/issues/21853
		for name, values := range resp.Header {
			name = strings.ToLower(name)
			for _, value := range values {
				headers = append(headers, fmt.Sprintf("%v: %v", name, value))
			}
		}

		ok := &reqCmdRepOK{
			V:       1,
			Cmd:     cmd.Cmd,
			UID:     cmd.UID,
			Us:      us,
			Code:    resp.StatusCode,
			Headers: headers,
			Payload: string(body),
		}
		return ok, nil
	}
}