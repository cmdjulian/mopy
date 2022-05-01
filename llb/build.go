package llb

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/moby/buildkit/client/llb"
	"github.com/moby/buildkit/exporter/containerimage/exptypes"
	"github.com/moby/buildkit/frontend/dockerfile/dockerfile2llb"
	"github.com/moby/buildkit/frontend/gateway/client"
	"github.com/pkg/errors"
	"gitlab.com/cmdjulian/buildkit-frontend-for-pythonv3/config"
)

const (
	LocalNameContext      = "context"
	LocalNameDockerfile   = "dockerfile"
	keyTarget             = "target"
	keyFilename           = "filename"
	keyCacheFrom          = "cache-from"
	defaultDockerfileName = "PyDockerfile.yaml"
	dockerignoreFilename  = ".dockerignore"
	buildArgPrefix        = "build-arg:"
	labelPrefix           = "label:"
	keyNoCache            = "no-cache"
	keyTargetPlatform     = "platform"
	keyMultiPlatform      = "multi-platform"
	keyImageResolveMode   = "image-resolve-mode"
	keyGlobalAddHosts     = "add-hosts"
	keyForceNetwork       = "force-network-mode"
	keyOverrideCopyImage  = "override-copy-image"
)

func Build(ctx context.Context, c client.Client) (*client.Result, error) {
	pyDockerConfig, err := GetPyDockerConfig(ctx, c)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get PyDockerfile")
	}

	dockerfile := PyDocker2LLB(pyDockerConfig)
	buildState, imgConfig, _, _ := dockerfile2llb.Dockerfile2LLB(ctx, []byte(dockerfile), dockerfile2llb.ConvertOpt{})

	def, err := buildState.Marshal(context.TODO())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal llb")
	}
	res, err := c.Solve(ctx, client.SolveRequest{
		Definition: def.ToPB(),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to resolve dockerfile")
	}
	ref, err := res.SingleRef()
	if err != nil {
		return nil, err
	}

	imageJsonByteConfig, err := json.Marshal(imgConfig)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal image config")
	}

	res.AddMeta(fmt.Sprintf("%s", exptypes.ExporterImageConfigKey), imageJsonByteConfig)
	res.SetRef(ref)

	return res, nil
}

func GetPyDockerConfig(ctx context.Context, c client.Client) (*config.Config, error) {
	opts := c.BuildOpts().Opts
	filename := opts[keyFilename]
	if filename == "" {
		filename = defaultDockerfileName
	}

	name := "load definition"
	if filename != "PyDockerfile" {
		name += " from " + filename
	}

	src := llb.Local(
		LocalNameDockerfile,
		llb.IncludePatterns([]string{filename}),
		llb.SessionID(c.BuildOpts().SessionID),
		llb.SharedKeyHint(defaultDockerfileName),
		dockerfile2llb.WithInternalName(name),
	)

	def, err := src.Marshal(context.TODO())
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal local source")
	}

	res, err := c.Solve(ctx, client.SolveRequest{
		Definition: def.ToPB(),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create solve request")
	}

	ref, err := res.SingleRef()
	if err != nil {
		return nil, err
	}

	var pyDockerfileYaml []byte
	pyDockerfileYaml, err = ref.ReadFile(ctx, client.ReadRequest{
		Filename: filename,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to read PyDockerfile")
	}

	cfg, err := config.NewFromBytes(pyDockerfileYaml)
	if err != nil {
		return nil, errors.Wrap(err, "error on getting parsing config")
	}

	return cfg, nil
}
