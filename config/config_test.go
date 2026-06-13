package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"github.com/tupicapp/common-go/config"
)

type sample struct {
	Name  string
	Value int
}

type ConfigSuite struct{ suite.Suite }

func TestConfigSuite(t *testing.T) {
	t.Parallel()
	suite.Run(t, new(ConfigSuite))
}

// write creates a JSON file in a fresh temp dir and returns its path.
func (s *ConfigSuite) write(name, body string) string {
	path := filepath.Join(s.T().TempDir(), name)
	s.Require().NoError(os.WriteFile(path, []byte(body), 0o600))
	return path
}

func (s *ConfigSuite) TestLoad_ReadsRequiredFile() {
	path := s.write("config.json", `{"name":"base","value":1}`)

	var got sample
	s.Require().NoError(config.Load(&got, path))
	s.Equal(sample{Name: "base", Value: 1}, got)
}

func (s *ConfigSuite) TestLoad_MissingRequiredFile_ReturnsError() {
	var got sample
	err := config.Load(&got, filepath.Join(s.T().TempDir(), "absent.json"))
	s.Require().Error(err)
}

func (s *ConfigSuite) TestLoad_OptionalOverrideMergesOverBase() {
	base := s.write("config.json", `{"name":"base","value":1}`)
	override := s.write("override.json", `{"value":2}`)

	var got sample
	s.Require().NoError(config.Load(&got, base, override))
	// Override wins for value; base survives for name.
	s.Equal(sample{Name: "base", Value: 2}, got)
}

func (s *ConfigSuite) TestLoad_MissingOptionalOverrideIsSkipped() {
	base := s.write("config.json", `{"name":"base","value":1}`)
	missing := filepath.Join(s.T().TempDir(), "override.json")

	var got sample
	// A missing optional override (not the first path) must not fail the load.
	s.Require().NoError(config.Load(&got, base, missing))
	s.Equal(sample{Name: "base", Value: 1}, got)
}

func (s *ConfigSuite) TestLoad_InvalidJSON_ReturnsError() {
	path := s.write("config.json", `{not valid json`)

	var got sample
	s.Require().Error(config.Load(&got, path))
}
