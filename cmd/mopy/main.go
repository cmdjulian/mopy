package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/frontend/dockerfile/dockerfile2llb"
	"github.com/moby/buildkit/frontend/gateway/grpcclient"
	"github.com/moby/buildkit/util/appcontext"
	"github.com/pkg/errors"
	"gitlab.com/cmdjulian/mopy/pkg/config"
	llbUtils "gitlab.com/cmdjulian/mopy/pkg/llb"
	"io"
	"os"
)

var filename string
var outputLLB bool
var outputDockerfile bool
var buildkit bool

func main() {
	flag.BoolVar(&outputLLB, "llb", false, "print llb to stdout")
	flag.BoolVar(&outputDockerfile, "dockerfile", false, "print equivalent Dockerfile to stdout")
	flag.BoolVar(&buildkit, "buildkit", true, "establish connection to buildkit and issue build")
	flag.StringVar(&filename, "filename", "Mopyfile.yaml", "the Mopyfile to build from")
	flag.Parse()

	if outputDockerfile {
		if err := printDockerfile(filename); err != nil {
			os.Exit(1)
		}
	}

	if outputLLB {
		if err := printLlb(filename, os.Stdout); err != nil {
			os.Exit(1)
		}
	}

	if buildkit {
		if err := grpcclient.RunFromEnvironment(appcontext.Context(), llbUtils.Build); err != nil {
			panic(err)
		}
	}
}

func printDockerfile(filename string) error {
	c, err := config.NewFromFilename(filename)
	if err != nil {
		return errors.Wrap(err, "opening Mopyfile")
	}
	dockerfile := llbUtils.Mopyfile2LLB(c)
	fmt.Println(dockerfile)

	return nil
}

func printLlb(filename string, out io.Writer) error {
	c, err := config.NewFromFilename(filename)
	if err != nil {
		return errors.Wrap(err, "opening Mopyfile")
	}
	dockerfile := llbUtils.Mopyfile2LLB(c)
	st, _, _, _ := dockerfile2llb.Dockerfile2LLB(context.TODO(), []byte(dockerfile), dockerfile2llb.ConvertOpt{})
	dt, err := st.Marshal(context.Background())
	if err != nil {
		return errors.Wrap(err, "marshaling llb state")
	}

	return llb.WriteTo(dt, out)
}
