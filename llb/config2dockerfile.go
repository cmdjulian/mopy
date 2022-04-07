package llb

import (
	"fmt"
	"gitlab.com/cmdjulian/buildkit-frontend-for-pythonv3/config"
	"gitlab.com/cmdjulian/buildkit-frontend-for-pythonv3/utils"
	"strings"
)

var defaultEnvs = map[string]string{
	"PIP_NO_CACHE_DIR":              "1",
	"PIP_DISABLE_PIP_VERSION_CHECK": "1",
	"PIP_USER":                      "1",
}

func PyDocker2LLB(c *config.Config) string {
	dockerfile := from(c)
	dockerfile += apt(c)
	dockerfile += env(utils.Union(defaultEnvs, c.Envs))
	dockerfile += installExternalDeps(c)
	dockerfile += installLocalDeps(c)
	dockerfile += installSshDeps(c)
	dockerfile += multistage(c)

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
		line += fmt.Sprintf("RUN pip install %s", depString)
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
		line += fmt.Sprintf("RUN --mount=type=ssh pip install %s", depString)
	}

	return line
}

func installLocalDeps(c *config.Config) string {
	line := "\n"
	deps := c.LocalDependencies()

	if len(deps) > 0 {
		for _, s := range deps {
			s = strings.TrimSuffix(s, "/")
			source := s + "/"
			s = utils.After(s, "/") + "/"
			target := "/tmp/" + s
			line += fmt.Sprintf("COPY %s %s\n", source, target)
			line += fmt.Sprintf("RUN pip install %s", target)
		}
	}

	return line
}

func multistage(c *config.Config) string {
	line := "\n"
	if strings.HasPrefix(c.PythonVersion, "3.9") {
		line += distroless39()
	} else {
		line += fallback(c)
	}

	if len(c.PipDependencies) > 0 {
		line += "\nCOPY --from=builder --chown=nonroot:nonroot /root/.local/ /home/nonroot/.local/"
	}

	if len(c.Envs) > 0 {
		line += env(c.Envs)
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
