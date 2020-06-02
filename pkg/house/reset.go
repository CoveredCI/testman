package house

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter"
	"github.com/FuzzyMonkeyCo/monkey/pkg/resetter/shell"
	"go.starlark.net/starlark"
)

func (rt *Runtime) reset(ctx context.Context) (err error) {
	rt.progress.Printf("Resetting system under test...\n")

	select {
	case <-time.After(txTimeout):
		err = errTXTimeout
	case err = <-rt.client.Snd(&fm.Clt{
		Msg: &fm.Clt_ResetProgress_{
			ResetProgress: &fm.Clt_ResetProgress{
				Status: fm.Clt_ResetProgress_started,
			}}}):
	}
	if err != nil {
		log.Println("[ERR]", err)
		return
	}

	var rsttr resetter.Interface
	for _, mdl := range rt.models {
		rsttr = mdl.GetResetter()
		break
	}

	stdout := newProgressWriter(rt.progress.Printf)
	stderr := newProgressWriter(rt.progress.Errorf)

	start := time.Now()
	err = rsttr.ExecReset(ctx, stdout, stderr, false)
	elapsed := time.Since(start).Nanoseconds()
	if err != nil {
		log.Println("[ERR] ExecReset:", err)
	}

	if err != nil {
		var reason []string
		if resetErr, ok := err.(*resetter.Error); ok {
			rt.progress.Errorf("Error resetting state!\n")
			reason = resetErr.Reason()
		} else {
			reason = strings.Split(err.Error(), "\n")
		}

		var err2 error
		select {
		case <-time.After(txTimeout):
			err2 = errTXTimeout
		case err2 = <-rt.client.Snd(&fm.Clt{
			Msg: &fm.Clt_ResetProgress_{
				ResetProgress: &fm.Clt_ResetProgress{
					Status:    fm.Clt_ResetProgress_failed,
					ElapsedNs: elapsed,
					Reason:    reason,
				}}}):
		}
		if err2 != nil {
			log.Println("[ERR]", err2)
			// nothing to continue on
		}
		return
	}

	select {
	case <-time.After(txTimeout):
		err = errTXTimeout
	case err = <-rt.client.Snd(&fm.Clt{
		Msg: &fm.Clt_ResetProgress_{
			ResetProgress: &fm.Clt_ResetProgress{
				Status:    fm.Clt_ResetProgress_ended,
				ElapsedNs: elapsed,
			}}}):
	}
	if err != nil {
		log.Println("[ERR]", err)
	}
	return
}

func newFromKwargs(modelerName string, r starlark.StringDict) (resetter.Interface, error) {
	const (
		tExecReset = "ExecReset"
		tExecStart = "ExecStart"
		tExecStop  = "ExecStop"
	)
	var (
		ok bool
		v  starlark.Value
		vv starlark.String
		t  string
		// TODO: other Resetter.s
		rsttr = &shell.Resetter{}
	)
	t = tExecStart
	if v, ok = r[t]; ok {
		delete(r, t)
		if vv, ok = v.(starlark.String); !ok {
			return nil, fmt.Errorf("%s(%s = ...) must be a string", modelerName, t)
		}
		rsttr.Start = vv.GoString()
	}
	t = tExecReset
	if v, ok = r[t]; ok {
		delete(r, t)
		if vv, ok = v.(starlark.String); !ok {
			return nil, fmt.Errorf("%s(%s = ...) must be a string", modelerName, t)
		}
		rsttr.Rst = vv.GoString()
	}
	t = tExecStop
	if v, ok = r[t]; ok {
		delete(r, t)
		if vv, ok = v.(starlark.String); !ok {
			return nil, fmt.Errorf("%s(%s = ...) must be a string", modelerName, t)
		}
		rsttr.Stop = vv.GoString()
	}
	if len(r) != 0 {
		return nil, fmt.Errorf("unexpected arguments to %s(): %s", modelerName, strings.Join(r.Keys(), ", "))
	}
	return rsttr, nil
}
