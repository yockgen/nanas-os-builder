// internal/config/config.go
package config

import (
    "encoding/json"
    "fmt"
    "os"
	"bytes"
	"go.uber.org/zap"
	"github.com/intel-innersource/os.linux.tiberos.os-curation-tool/internal/validate"
)

// Config represents your JSON schema.
type Config struct {
    Distro   string   		`json:"distro"`
    Version  string  		`json:"version"`
    Arch     string  		`json:"arch"`
    Packages []string 		`json:"packages"`
	Immutable bool   		`json:"immutable"`
	Output  string   		`json:"output"`
	Kernel  KernelConfig   	`json:"kernel"`
}

// KernelConfig holds the nested “kernel” object.
type KernelConfig struct {
    Version string `json:"version"`
    Cmdline string `json:"cmdline"`
}

func Load(path string) (*Config, error) {
    logger := zap.L().Sugar()
	
	data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    // validate raw JSON against schema
    if err := validate.ValidateJSON(data); err != nil {
        return nil, fmt.Errorf("validation error: %w", err)
    }
    // unmarshal into typed struct
    var cfg Config
    dec := json.NewDecoder(bytes.NewReader(data))
    dec.DisallowUnknownFields()
    if err := dec.Decode(&cfg); err != nil {
        return nil, fmt.Errorf("parsing config: %w", err)
    }

	logger.Infof("loaded config: \n%s", string(data))
    return &cfg, nil
}