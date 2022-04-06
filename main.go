package main

import (
	"context"
	"flag"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/dockerfile/dockerfile2llb"
	"github.com/moby/buildkit/frontend/gateway/grpcclient"
	"github.com/moby/buildkit/util/appcontext"
	"github.com/pkg/errors"
	"gitlab.com/cmdjulian/buildkit-frontend-for-pythonv3/config"
	pydocker "gitlab.com/cmdjulian/buildkit-frontend-for-pythonv3/llb"
	"io"
	"os"
)

var filename string
var graph bool

func main() {
	flag.BoolVar(&graph, "graph", false, "output a graph and exit")
	flag.StringVar(&filename, "filename", "PyDockerfile.yaml", "the PyDockerfile to build from")
	flag.Parse()

	if graph {
		if err := printLLB(filename, os.Stdout); err != nil {
			os.Exit(1)
		}
		os.Exit(0)
	}

	if err := grpcclient.RunFromEnvironment(appcontext.Context(), pydocker.Build); err != nil {
		panic(err)
	}

}

func printLLB(filename string, out io.Writer) error {
	c, err := config.NewFromFilename(filename)
	if err != nil {
		return errors.Wrap(err, "opening PyDockerfile")
	}
	dockerfile := pydocker.PyDocker2LLB(c)
	st, _, _, _ := dockerfile2llb.Dockerfile2LLB(context.TODO(), []byte(dockerfile), dockerfile2llb.ConvertOpt{})
	dt, err := st.Marshal(context.Background())
	if err != nil {
		return errors.Wrap(err, "marshaling llb state")
	}

	return llb.WriteTo(dt, out)
}
