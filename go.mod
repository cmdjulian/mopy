module gitlab.com/cmdjulian/buildkit-frontend-for-pythonv3

go 1.16

require (
	github.com/docker/docker v20.10.14+incompatible // indirect
	github.com/moby/buildkit v0.10.0
	github.com/opencontainers/image-spec v1.0.2 // indirect
	github.com/pkg/errors v0.9.1
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/Sirupsen/logrus v1.8.1 => github.com/sirupsen/logrus v1.8.1

require (
	github.com/containerd/containerd v1.6.2
	github.com/docker/distribution v2.8.1+incompatible // indirect
	golang.org/x/crypto v0.0.0-20220331220935-ae2d96664a29 // indirect
	golang.org/x/sys v0.0.0-20220403205710-6acee93ad0eb // indirect
	google.golang.org/genproto v0.0.0-20220401170504-314d38edb7de // indirect
)
