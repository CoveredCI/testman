package house

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/FuzzyMonkeyCo/monkey/pkg/internal/fm"
	"github.com/FuzzyMonkeyCo/monkey/pkg/modeler"
	"github.com/FuzzyMonkeyCo/monkey/pkg/ui/ci"
	"github.com/FuzzyMonkeyCo/monkey/pkg/ui/cli"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// grpcHost isn't const so ldflags can rewrite it
var grpcHost = "do.dev.fuzzymonkey.co:7077"

func (rt *Runtime) dial(ctx context.Context, apiKey string) (
	closer func(),
	err error,
) {
	log.Println("[NFO] dialing", grpcHost)
	var conn *grpc.ClientConn
	if conn, err = grpc.DialContext(ctx, grpcHost,
		grpc.WithBlock(),
		grpc.WithTimeout(4*time.Second),
		grpc.WithInsecure(),
	); err != nil {
		log.Println("[ERR]", err)
		return
	}

	ctx = metadata.AppendToOutgoingContext(ctx,
		"ua", rt.binTitle,
		"apiKey", apiKey,
	)

	if rt.client, err = fm.NewFuzzyMonkeyClient(conn).Do(ctx); err != nil {
		log.Println("[ERR]", err)
		return
	}
	closer = func() {
		if err := rt.client.CloseSend(); err != nil {
			log.Println("[ERR]", err)
		}
		if err := conn.Close(); err != nil {
			log.Println("[ERR]", err)
		}
	}
	return
}

func (rt *Runtime) newProgressWithNtensity(ntensity uint32) {
	if rt.logLevel != 0 {
		rt.progress = &ci.Progresser{}
	} else {
		rt.progress = &cli.Progresser{}
	}
	rt.testingCampaingStart = time.Now()
	rt.progress.MaxTestsCount(10 * ntensity)
}

func (rt *Runtime) Fuzz(ctx context.Context, ntensity uint32, apiKey string) (err error) {
	var closer func()
	if closer, err = rt.dial(ctx, apiKey); err != nil {
		return
	}
	defer closer()

	var mdl modeler.Interface
	for _, mdl = range rt.models {
		break
	}

	rt.newProgressWithNtensity(ntensity)
	ctx = context.WithValue(ctx, "UserAgent", rt.binTitle)

	log.Printf("[DBG] sending initial msg")
	if err = rt.client.Send(&fm.Clt{
		Msg: &fm.Clt_Fuzz_{
			Fuzz: &fm.Clt_Fuzz{
				EIDs:      rt.eIds,
				EnvRead:   rt.envRead,
				Model:     mdl.ToProto(),
				ModelKind: fm.Clt_Fuzz_OpenAPIv3,
				Ntensity:  ntensity,
				Resetter:  mdl.GetResetter().ToProto(),
				// FIXME: seeding
				Seed:  []byte{42, 42, 42},
				Tags:  rt.tags,
				Usage: os.Args,
			}}}); err != nil {
		log.Println("[ERR]", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			log.Println("[ERR]", err)
		default:
		}
		if err != nil {
			break
		}

		log.Printf("[DBG] receiving msg...")
		var srv *fm.Srv
		if srv, err = rt.client.Recv(); err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			log.Println("[ERR]", err)
			break
		}

		switch srv.GetMsg().(type) {
		case *fm.Srv_Call_:
			log.Println("[NFO] handling fm.Srv_Call_")
			cll := srv.GetCall()
			if err = rt.call(ctx, cll); err != nil {
				break
			}
			log.Println("[NFO] done handling fm.Srv_Call_")
		case *fm.Srv_Reset_:
			log.Println("[NFO] handling fm.Srv_Reset_")
			if err = rt.reset(ctx); err != nil {
				break
			}
			log.Println("[NFO] done handling fm.Srv_Reset_")
		default:
			err = fmt.Errorf("unhandled srv msg %T: %+v", srv.GetMsg(), srv)
			log.Println("[ERR]", err)
			break
		}

		if err2 := rt.recvFuzzProgress(); err2 != nil {
			if err == nil {
				err = err2
			}
			break
		}
	}

	log.Println("[DBG] server dialog ended, cleaning up...")
	if err2 := mdl.GetResetter().Terminate(ctx, nil); err2 != nil {
		log.Println("[ERR]", err2)
		if err == nil {
			err = err2
		}
	}
	if err2 := rt.progress.Terminate(); err2 != nil {
		log.Println("[ERR]", err2)
		if err == nil {
			err = err2
		}
	}

	if err == nil {
		err = rt.campaignSummary()
	}
	return
}
