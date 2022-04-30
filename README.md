# Buildkit Frontend for Python

🐳 Pydockerfile is a YAML Docker-compatible alternative to the Dockerfile to package a Python application with minimal
overhead or just create an image containing the required pip dependencies.  
No need to know or learn Docker!

## Installation as cmd

```bash
$ go get -u gitlan.com/cmdjulian/pydockerfile
```

## Usage

### Pydockerfile config file

create a Pydockerfile.yaml:

```yaml
#syntax=cmdjulian/pydockerfile

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

`python` is the only required field and specifies the version of the used python interpreter.

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

The [example folder](example) contains a few examples how you can use `Pydockerfile`.

### Build Pydockerfile using docker build

```
$ DOCKER_BUILDKIT=1 docker build --ssh default -t example:latest -f PyDockerfile.yaml .
```

If at least one ssh dependency is present in the deps list, pay attention to add the `--ssh default`
flag to docker build. Also make sure, that your ssh-key is loaded inside the ssh agent.

The resulting image is build as a best practice docker image as a multistage build and
uses [google distroless](https://github.com/GoogleContainerTools/distroless) image as final base image. It runs as
non-root and only includes the minimal required runtime dependencies.

### Build Pydockerfile with builtctl

using as buildkit frontend.

```bash
buildctl build \
    --frontend=gateway.v0 \
    --opt source=cmdjulian/pydockerfile \
    --ssh default \
    --local context=. \
    --local dockerfile=. \
    --output type=docker,name=example:latest \
| docker load
```

## Run container

The build image can be run like any other container:

```bash
$ docker run --rm example:latest
```

## Credits
- https://earthly.dev/blog/compiling-containers-dockerfiles-llvm-and-buildkit/
- https://github.com/moby/buildkit/blob/master/docs/merge%2Bdiff.md
- https://github.com/moby/buildkit/blob/master/frontend/dockerfile/docs/syntax.md
