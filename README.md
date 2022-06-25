# `mopy` - a Buildkit Frontend for Python

🐳 `mopy` is a YAML Docker-compatible alternative to the Dockerfile to package a Python application with minimal
overhead. Mopy can also create base images containing a certain set of dependencies. To run mopy no installation is
required, as it is seemingly integrated and run by [buildkit](https://github.com/moby/buildkit)
(or [docker](https://github.com/docker/buildx)). Docker build is therefor taking care of getting and running mopy.  
To make use of mopy, you don't have to be a docker pro!

## Mopyfile

`Mopyfile` is the equivalent of `Dockerfile` for mopy. It is based on `yaml` and assembles a python specific dsl.
Start by creating a `Mopyfile.yaml` file:

[//]: # (@formatter:off)
```yaml
#syntax=cmdjulian/mopy                                 # [1] Enable Mopy syntax

apiVersion: v1                                         # [2] Mopyfile api version
python: 3.9.2                                          # [3] python interpreter version
build-deps:                                            # [4] additional 'apt' packages installed before build
  - libopenblas-dev
  - gfortran
  - build-essential
envs:                                                  # [5] environment variables available in build stage and in the final image
  MYENV: envVar1
pip:                                                   # [6] pip dependencies to install
  - numpy==1.22                                          # use version 1.22 of 'numpy'
  - slycot                                               # use version 'latest' of 'slycot'
  - git+https://github.com/moskomule/anatome.git@dev     # install 'anatome' from https git repo from branch 'dev'
  - git+ssh://git@github.com/RRZE-HPC/pycachesim.git     # install 'pycachesim' from ssh repo on 'default' branch
  - ./my_local_pip/                                      # use local fs folder from working directory (has to start with ./ )
  - ./requirements.txt                                   # include pip packages from requirements.txt file from working directory (has to start with ./ )
sbom: true                                             # [7] include pip dependencies as label
labels:                                                # [8] additional labels to include in final image
  foo: bar
project: my-python-app/                                # [9] include executable python file(s)
```
[//]: # (@formatter:on)

The most important part of the file is the first line `#syntax=cmdjulian/mopy`. It tells docker buildkit to use the
mopy frontend. The frontend is compatible with linux, windows and mac. It also supports various cpu architectures.
Currently `i386`, `amd64`, `arm/v6`, `arm/v7`, `arm64/v8` are supported. Buildkit automatically picks the right version
for you from dockerhub.

Availabe configuration options are listed in the table below.

|     | required | description                                                                                                                                                                                                                                                                              | default | type                    |
|-----|----------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------|-------------------------|
| 1   | yes      | instruct Docker to use `Mopyfile` syntax for parsing this file                                                                                                                                                                                                                           | -       | docker syntax directive |
| 2   | no       | api version of `Mopy` file format. This is mainly due to future development to prevent incompatibilities                                                                                                                                                                                 | v1      | enum: [`v1`]            |
| 3   | yes      | the python interpreter version to use. Versions format is thereby: `3`, `3.9` or `3.9.1`                                                                                                                                                                                                 | -       | string                  |
| 4   | no       | additional `apt` packages to install before staring the build. These are not part of the final image                                                                                                                                                                                     | -       | string[]                |
| 5   | no       | additional environment variables. These are present in the build and in the run stage                                                                                                                                                                                                    | -       | map\[string]\[string]   |
| 6   | no       | list of pip dependencies to install                                                                                                                                                                                                                                                      | -       | string[]                |
| 7   | no       | add an sbom label. For details see the [sbom](#sbom) section                                                                                                                                                                                                                             | false   | boolean                 |
| 8   | no       | additional labels to add to the final image. These have precedence over automatically added                                                                                                                                                                                              | -       | map\[string]\[string]   |
| 9   | no       | relative path to a `Python` file or folder. If the path points to a folder, the folder has to contain a `main.py` file. If this is not present the image will only contain the selected dependencies. If this is present, the project or file gets set as entrypoint for the final image | -       | string                  |

The [example folder](example) contains a few examples how you can use `mopy`.

### sbom

when the `sbom` field is set to `true`, a label called `mopy.sbom` contains a `json` representation of the supplied
dependencies. Be aware, that when you use a dependency that contains basic in url auth credentials, these are getting
exposed in the label. A workaround for that, is to use that sort of url in a dedicated `requirements.txt` file, as their
content gets not included in the `json` structure. Or better use ssh links.
Consider the following `Mopyfile`:

```yaml
#syntax=cmdjulian/mopy:v1

python: 3.10
pip:
  - numpy==1.22
  - catt
  - git+https://github.com/moskomule/anatome.git@dev
  - git+ssh://git@github.com/RRZE-HPC/pycachesim.git
  - ./requirements.txt
sbom: true
```

This yields the following `json` structure:

```json
[
  "numpy==1.22",
  "catt",
  "git+https://github.com/moskomule/anatome.git@dev",
  "git+ssh://git@github.com/RRZE-HPC/pycachesim.git",
  "./requirements.txt"
]
```

## Build `Mopyfile`

`Mopyfile` can be build with every docker buildkit compatible cli. The following are a few examples:

#### docker:

```bash
DOCKER_BUILDKIT=1 docker build --ssh default -t example:latest -f Mopyfile.yaml .
```

#### nerdctl:

```bash
nerdctl build --ssh default -t example:latest -f Mopyfile.yaml .
```

#### buildctl:

```bash
buildctl build \
--frontend=gateway.v0 \
--opt source=cmdjulian/mopy \
--ssh default \
--local context=. \
--local dockerfile=. \
--output type=docker,name=example:latest \
| docker load
```

The resulting image is build as a best practice docker image and employs a multistage build- It
uses [google distroless](https://github.com/GoogleContainerTools/distroless) image as final base image. It runs as
non-root user and only includes the minimal required runtime dependencies.

### SSH dependencies

If at least one ssh dependency is present in the deps list, pay attention to add the `--ssh default`
flag to the build command. Also make sure, that your ssh-key is loaded inside the ssh agent.  
If you receive an error `invalid empty ssh agent socket, make sure SSH_AUTH_SOCK is set` your SSH agent is not running
or improperly set up. You can start or configure it and adding your ssh key by executing:

```bash
eval `ssh-agent`
ssh-add /path/to/ssh-key
```

The `ssh` flag is only required if you're including a ssh dependency. If no ssh dependency is present, the ssh flag can
be omitted.

## Run a container from the built image

The built image can be run like any other container:

```bash
$ docker run --rm example:latest
```

## mopy development

### Installation as cmd

```bash
$ go install gitlan.com/cmdjulian/mopy
```

### Arguments

The following arguments are supported running the frontend:

| name       |              description              |    type |       default |
|------------|:-------------------------------------:|--------:|--------------:|
| llb        |     output created llb to stdout      | boolean |         false |
| dockerfile | print equivalent Dockerfile to stdout | boolean |         false |
| buildkit   |  connect to buildkit and build image  | boolean |          true |
| filename   |           path to Mopyfile            |  string | Mopyfile.yaml |

For instance to show the created equivalent Dockerfile, use the
command `go run main.go -buildkit=false -dockerfile=true -filename=example/full/Mopyfile.yaml`.

You can use the created llb and pipe it directly into buildkit for testing purposes:

```bash
docker run --rm --privileged -d --name buildkit moby/buildkit
export BUILDKIT_HOST=docker-container://buildkit
go run cmd/mopy/main.go -llb=true -buildkit=false -filename=example/full/Mopyfile.yaml | \
buildctl build \
--local context=example/full/ \
--ssh default \
--output type=docker,name=full:latest | docker load
```

## Credits

- https://earthly.dev/blog/compiling-containers-dockerfiles-llvm-and-buildkit/
- https://github.com/moby/buildkit/blob/master/docs/merge%2Bdiff.md
- https://github.com/moby/buildkit/blob/master/frontend/dockerfile/docs/syntax.md
