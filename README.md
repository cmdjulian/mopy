# `mopy` - a Buildkit Frontend for Python

🐳 `mopy` is a YAML Docker-compatible alternative to the Dockerfile to package a Python application with minimal
overhead. Mopy can also create base images containing a certain set of dependencies. To run mopy no installation is
required, as it is seemingly integrated into docker build and therefore docker build is taking care of getting and
running it. To make use of mopy, you don't have to be a docker pro!

## Mopyfile

`Mopyfile` is the equivalent of `Dockerfile` for mopy. It is based on `yaml` and assembles a python specific dsl.
Start by creating a `Mopyfile.yaml` file:

```yaml
#syntax=cmdjulian/mopy

apiVersion: v1
python: 3.9.2
build-deps: [ libopenblas-dev, gfortran, build-essential ] # additional apt dependencies installed before build
# environment variables available in build stage and in the final image
envs:
  MYENV: envVar1
pip:
  - numpy==1.22                                            # use version 1.22 of numpy
  - slycot                                                 # use latest version of slycot
  - git+https://github.com/moskomule/anatome.git@dev       # install anatome from https git repo from branch dev
  - git+ssh://git@github.com/RRZE-HPC/pycachesim.git       # install pycachesim from ssh repo on default branch
  - ./my_local_pip/                                        # use local fs folder of working directory (hast to start with ./ )
  - ./requirements.txt                                     # installation from requirements.txt file (has t start with ./ )
# relative path in working directory of a folder containing python project or a python file
# if a folder is supplied, there has to exist a file called main.py in it
project: my-python-app/
```

The most important part of the file is the first line `#syntax=cmdjulian/mopy`. It tells docker buildkit to use the
mopy frontend. The frontend is compatible with linux, windows and mac. It also supports various cpu architectures.
Currently i386, amd64, arm/v6, arm/v7, arm64/v8 are supported. Buildkit automatically picks the right version for you
from dockerhub.

Except for the syntax directive, `python` is the only required field and specifies the version of the used version of
the python interpreter.

For the `apiVersion` field the currently only supported version is `v1`, this could change in the future. If you omit
the version field, `v1` is assumed.

`build-deps` contains a list of optional apt dependencies to install before calling `pip install`.

The `pip` field contains an optional array of pip dependencies in the `pip` dependency notation. Additionally, a
relative path to a `requirements.txt` is supported. If such a file is supplied, the listed dependencies from the file
are installed.

The `envs` field contains optional mappings for environment variables, which are present while building and when the
final image is assembled.

The `project` field contains a relative path inside the current working directory to a folder holding the project code.
This project folder has to contain a `main.py` file. Also, a path to a single python file is supported. Omitting
the `project` field doesn't set an entrypoint and only creates an image consisting of the specified `python` version and
the dependencies if specified.

The [example folder](example) contains a few examples how you can use `mopy`.

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
$ go get -u gitlan.com/cmdjulian/mopy
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
go run main.go -llb=true -buildkit=false -filename=example/full/Mopyfile.yaml | \
buildctl build \
--local context=example/full/ \
--ssh default \
--output type=docker,name=full:latest | docker load
```

## Credits

- https://earthly.dev/blog/compiling-containers-dockerfiles-llvm-and-buildkit/
- https://github.com/moby/buildkit/blob/master/docs/merge%2Bdiff.md
- https://github.com/moby/buildkit/blob/master/frontend/dockerfile/docs/syntax.md
