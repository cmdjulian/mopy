package config

import (
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/cmdjulian/mopy/pkg/utils"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"regexp"
	"strings"
)

var httpPattern = regexp.MustCompile(`^http(s)?://`)
var gitHttpPattern = regexp.MustCompile(`^git\+http(s)?://`)

// NewFromFilename returns a new config from a filename
func NewFromFilename(filename string) (*Config, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, errors.Wrap(err, "opening file")
	}
	defer f.Close()
	contents, err := ioutil.ReadAll(f)
	if err != nil {
		return nil, errors.Wrap(err, "reading config file")
	}

	return NewFromBytes(contents)
}

func NewFromBytes(b []byte) (*Config, error) {
	c := &Config{}
	if err := yaml.Unmarshal(b, c); err != nil {
		return nil, errors.Wrap(err, "unmarshal config")
	}

	return c, c.Validate()
}

type Config struct {
	ApiVersion      string            `default:"v1" yaml:"apiVersion"`
	PythonVersion   string            `default:"3.9" yaml:"python"`
	Apt             []string          `yaml:"build-deps"`
	Envs            map[string]string `yaml:"envs"`
	Indices         []Index           `yaml:"indices"`
	PipDependencies []string          `yaml:"pip"`
	Project         string            `yaml:"project"`
	Labels          map[string]string `yaml:"labels"`
	Sbom            *bool             `default:"true" yaml:"sbom"`
}

type Index struct {
	Url      string `yaml:"url"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Trust    bool   `default:"false" yaml:"trust"`
}

func (c *Config) Validate() error {
	if c.ApiVersion != "" && c.ApiVersion != "v1" {
		return fmt.Errorf("unknown version %s. Known versions: 'v1'", c.ApiVersion)
	}

	if c.PythonVersion == "" {
		return errors.New("empty is not a valid Python Version")
	}
	match, _ := regexp.MatchString("^[2-9](\\.\\d{1,2})?(\\.\\d{1,2})?$", c.PythonVersion)
	if !match {
		return fmt.Errorf("%s is not a valid Python Version", c.PythonVersion)
	}

	invalidPaths := c.dependenciesFilteredByPrefix("/")
	if len(invalidPaths) > 0 {
		return fmt.Errorf("local paths can only be relative, found: %s", strings.Join(invalidPaths, ", "))
	}

	if c.Project != "" {
		if strings.HasPrefix(c.Project, "/") {
			return fmt.Errorf("project path can't be absolute, has to be relative, found: %s", c.Project)
		}
		if !strings.HasPrefix(c.Project, "./") {
			c.Project = "./" + c.Project
		}
	}

	return nil
}

func (c *Config) MaskedDependencies() []string {
	dependencies := c.PipDependencies

	for i, dependency := range dependencies {
		switch {
		case httpPattern.MatchString(dependency):
			{
				ref, err := url.Parse(dependency)
				if err != nil {
					log.Fatal(err)
				}

				ref.User = nil
				dependencies[i] = ref.String()
			}

		case gitHttpPattern.MatchString(dependency):
			{
				ref, err := url.Parse(dependency[4:])
				if err != nil {
					log.Fatal(err)
				}

				ref.User = nil
				dependencies[i] = "git+" + ref.String()
			}
		}
	}

	return dependencies
}

func (c *Config) PyPiDependencies() []string {
	local := c.LocalDependencies()
	ssh := c.SshDependencies()
	http := c.HttpDependencies()
	combined := append(append(http, ssh...), local...)

	return utils.Difference(combined, c.PipDependencies)
}

func (c *Config) HttpDependencies() []string {
	http := c.dependenciesFilteredByPrefix("git+http://")
	https := c.dependenciesFilteredByPrefix("git+https://")
	combined := append(http, https...)

	return utils.RemoveDuplicate(combined)
}

func (c *Config) SshDependencies() []string {
	return c.dependenciesFilteredByPrefix("git+ssh://")
}

func (c *Config) LocalDependencies() []string {
	return c.dependenciesFilteredByPrefix("./")
}

func (c *Config) dependenciesFilteredByPrefix(filter string) []string {
	var filtered []string

	for i := range c.PipDependencies {
		if strings.HasPrefix(c.PipDependencies[i], filter) {
			filtered = append(filtered, c.PipDependencies[i])
		}
	}

	return filtered
}
