package collector

import (
	"fmt"
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

type HostConfig struct {
	Host       string    `yaml:"host"`
	Ip         string    `yaml:"ip"`
	Port       string    `yaml:"port"`
	User       string    `yaml:"user"`
	Password   string    `yaml:"password"`
	Key        string    `yaml:"private_key"`
	Passphrase string    `yaml:"passphrase"`
	Proxy      string    `yaml:"proxy_host"`
	Exporter   Exporters `yaml:"exporters"`
}

type Config struct {
	Hosts []HostConfig `yaml:"hosts"`
}

var HostMap map[string]HostConfig

func LoadConfig(config_file string) {
	yamlFile, err := os.ReadFile(config_file)
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

	HostMap = make(map[string]HostConfig)
	for _, host := range config.Hosts {
		if host.Port == "" {
			host.Port = "22"
		}
		if host.Ip == "" {
			host.Ip = host.Host
		}
		host.Password, _ = Decrypt(host.Password)
		host.Passphrase, _ = Decrypt(host.Passphrase)
		HostMap[host.Host] = host
	}
}

func GetConfig(targetHost string) HostConfig {
	return HostMap[targetHost]
}

func Collect(targetHost string, exporter string) string {
	switch exporter {
	case "kafka_exporter":
		return CollectKafkaMetrics(targetHost)
	default:
		return "# metrics not found"
	}
}

func CollectKafkaMetrics(targetHost string) string {
	output := make(chan string)
	kafkaExporterOption := GetConfig(targetHost).Exporter.KafkaExporter
	if len(kafkaExporterOption.KafkaServer) == 0 {
		slog.Error("kafka: client has run out of available brokers")
		return "# kafka: client has run out of available brokers"
	}
	var kafkaServers string
	for _, v := range kafkaExporterOption.KafkaServer {
		kafkaServers += "--kafka.server=" + v + " "
	}
	if kafkaExporterOption.Port == "" {
		kafkaExporterOption.Port = "9308"
	}
	command := fmt.Sprintf("(.promethues_exporter/kafka_exporter --web.listen-address=:%[2]s %[1]s > /dev/null 2>&1 & sleep 3; curl -s http://localhost:%[2]s/metrics) && pkill kafka_exporter", kafkaServers, kafkaExporterOption.Port)
	go CollectBySsh(targetHost, command, output)
	return <-output
}
