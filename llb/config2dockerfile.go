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
	dockerfile = env(dockerfile)
	dockerfile = installExternalDeps(dockerfile, c)
	dockerfile = installLocalDeps(dockerfile, c)
	dockerfile = installSshDeps(dockerfile, c)
	dockerfile = dockerfile + "\n"
	dockerfile = multistage(dockerfile, c)

	return dockerfile
}

func multistage(state string, c *config.Config) string {
	if strings.HasPrefix(c.PythonVersion, "3.9") {
		state = distroless39(state)
	} else {
		state = fallback(state, c)
	}

	state += "COPY --from=builder --chown=nonroot:nonroot /root/.local/ /home/nonroot/.local/\n"
	return state
}

func distroless39(state string) string {
	state += "FROM gcr.io/distroless/python3:nonroot@sha256:59e6b46683dc13d0267729efc94e04cefb85e29a78fa64109685e6c7afdc95eb\n"
	state += "COPY --from=builder --link --chown=nonroot:nonroot /root/.local/ /home/nonroot/.local/\n"
	return state
}

func fallback(state string, c *config.Config) string {
	state += fmt.Sprintf("FROM python:%s-slim\n", c.PythonVersion)
	state += "RUN useradd --uid=65532 --user-group --home-dir=/home/nonroot --create-home nonroot\n"
	state += "USER 65532:65532\n"

	return state
}

func from(c *config.Config) string {
	return fmt.Sprintf("FROM python:%s AS builder\n", c.PythonVersion)
}

func env(state string) string {
	state += "ENV "
	for key, value := range defaultEnvs {
		state = state + fmt.Sprintf("%s=%s ", key, value)
	}

	return state + "\n"
}

func installExternalDeps(state string, c *config.Config) string {
	deps := append(c.PyPiDependencies(), c.HttpDependencies()...)

	if len(deps) > 0 {
		depString := strings.Join(deps, " ")
		state += fmt.Sprintf("RUN pip install %s\n", depString)
	}

	return state
}

func installSshDeps(state string, c *config.Config) string {
	deps := c.SshDependencies()

	if len(deps) > 0 {
		depString := strings.Join(deps, " ")
		state += "ENV GIT_SSH_COMMAND=\"ssh -o StrictHostKeyChecking=no\"\n"
		state += "RUN apt update && apt install git-lfs && git lfs install\n"
		state += fmt.Sprintf("RUN --mount=type=ssh pip install %s\n", depString)
	}

	return state
}

func installLocalDeps(state string, c *config.Config) string {
	deps := c.LocalDependencies()

	if len(deps) > 0 {
		for _, s := range deps {
			s = strings.TrimSuffix(s, "/")
			source := s + "/"
			s = utils.After(s, "/") + "/"
			target := "/tmp/" + s
			state += fmt.Sprintf("COPY %s %s\n", source, target)
			state += fmt.Sprintf("RUN pip install %s\n", target)
		}
	}

	return state
}
