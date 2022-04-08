package llb

import (
	"fmt"
	"gitlab.com/cmdjulian/buildkit-frontend-for-pythonv3/config"
	"gitlab.com/cmdjulian/buildkit-frontend-for-pythonv3/utils"
	"strings"
)

var defaultEnvs = map[string]string{
	"PIP_DISABLE_PIP_VERSION_CHECK": "1",
	"PIP_NO_WARN_SCRIPT_LOCATION":   "0",
	"PIP_USER":                      "1",
}

const cacheMount = "--mount=type=cache,target=/root/.cache"

func PyDocker2LLB(c *config.Config) string {
	dockerfile := buildStage(c)
	dockerfile += runStage(c)

	return dockerfile
}

func buildStage(c *config.Config) string {
	dockerfile := from(c)
	dockerfile += apt(c)
	dockerfile += env(utils.Union(defaultEnvs, c.Envs))
	dockerfile += installExternalDeps(c)
	dockerfile += installLocalDeps(c)
	dockerfile += installSshDeps(c)

	return dockerfile
}

func from(c *config.Config) string {
	return fmt.Sprintf("FROM python:%s AS builder\n", c.PythonVersion)
}

func apt(c *config.Config) string {
	line := "\n"

	if len(c.Apt) > 0 {
		line += "RUN apt update && apt install -y "
		for _, apt := range c.Apt {
			line += fmt.Sprintf("%s ", apt)
		}
	}

	return line
}

func env(envs map[string]string) string {
	line := "\nENV"
	for key, value := range envs {
		line += fmt.Sprintf(" %s=%s", key, value)
	}

	return line
}

func installExternalDeps(c *config.Config) string {
	line := "\n"
	deps := append(c.PyPiDependencies(), c.HttpDependencies()...)

	if len(deps) > 0 {
		depString := strings.Join(deps, " ")
		line += fmt.Sprintf("RUN %s pip install %s", cacheMount, depString)
	}

	return line
}

func installSshDeps(c *config.Config) string {
	line := "\n"
	deps := c.SshDependencies()

	if len(deps) > 0 {
		depString := strings.Join(deps, " ")
		line += "RUN apt update && apt install git-lfs\n"
		line += "ENV GIT_SSH_COMMAND=\"ssh -o StrictHostKeyChecking=no\"\n"
		line += fmt.Sprintf("RUN %s --mount=type=ssh pip install %s", cacheMount, depString)
	}

	return line
}

func installLocalDeps(c *config.Config) string {
	line := ""
	deps := c.LocalDependencies()

	if len(deps) > 0 {
		for _, s := range deps {
			if strings.HasSuffix(s, "/requirements.txt") {
				target := "/tmp/requirements.txt"
				line += fmt.Sprintf("\nRUN %s --mount=type=bind,source=%s,target=%s pip install -r %s", cacheMount, s, target, target)
			} else {
				s = strings.TrimSuffix(s, "/")
				source := s + "/"
				s = utils.After(s, "/") + "/"
				target := "/tmp/" + s
				line += fmt.Sprintf("COPY %s %s\n", source, target)
				line += fmt.Sprintf("RUN %s pip install %s", cacheMount, target)
			}
		}
	}

	return line
}

func runStage(c *config.Config) string {
	line := "\n"
	if strings.HasPrefix(c.PythonVersion, "3.9") {
		line += distroless39()
	} else {
		line += fallback(c)
	}

	if len(c.Envs) > 0 {
		line += env(c.Envs)
	}

	if len(c.PipDependencies) > 0 {
		line += "\nCOPY --from=builder --chown=nonroot:nonroot /root/.local/ /home/nonroot/.local/"
	}

	if c.Project != "" {
		line += project(c)
	}

	return line
}

func distroless39() string {
	return "FROM gcr.io/distroless/python3:nonroot@sha256:59e6b46683dc13d0267729efc94e04cefb85e29a78fa64109685e6c7afdc95eb"
}

func fallback(c *config.Config) string {
	line := fmt.Sprintf("FROM python:%s-slim\n", c.PythonVersion)
	line += "RUN useradd --uid=65532 --user-group --home-dir=/home/nonroot --create-home nonroot\n"
	line += "USER 65532:65532"

	return line
}

func project(c *config.Config) string {
	line := "\n"

	project := strings.TrimSuffix(c.Project, "/")
	source := "/home/nonroot/" + utils.After(project, "/")
	line += fmt.Sprintf("COPY --chown=nonroot:nonroot %s %s\n", c.Project, source)
	line += "ENTRYPOINT [ \"python\", \"-u\" ]\n"

	if strings.HasSuffix(c.Project, ".py") {
		line += "WORKDIR /home/nonroot\n"
		line += fmt.Sprintf("CMD [ \"%s\" ]", source)
	} else {
		line += fmt.Sprintf("WORKDIR %s\n", source)
		line += "CMD [ \"main.py\" ]"
	}

	return line
}
