package collector

import (
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
)

type Exporters struct {
	KafkaExporter KafkaExporterOption `yaml:"kafka_exporter"`
}

type KafkaExporterOption struct {
	Port        string   `yaml:"port"`
	KafkaServer []string `yaml:"kafka.server"`
}

type JumpHost struct {
	Addr       string `yaml:"address"`
	User       string `yaml:"user"`
	Password   string `yaml:"password"`
	Key        string `yaml:"private_key"`
	Passphrase string `yaml:"passphrase"`
}

type Target struct {
	Name       string     `yaml:"name"`
	Addr       string     `yaml:"address"`
	User       string     `yaml:"user"`
	Password   string     `yaml:"password"`
	Key        string     `yaml:"private_key"`
	Passphrase string     `yaml:"passphrase"`
	JumpHosts  []JumpHost `yaml:"jump_hosts"`
	Exporter   Exporters  `yaml:"exporters"`
}

type Config struct {
	Targets []Target `yaml:"targets"`
}

var TargetMap map[string]Target

func LoadConfig(configFile string) {
	yamlFile, err := os.ReadFile(configFile)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(0)
	}
	var config Config
	err = yaml.Unmarshal(yamlFile, &config)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(0)
	}

	TargetMap = make(map[string]Target)
	for _, t := range config.Targets {
		TargetMap[t.Name] = t
	}
}
