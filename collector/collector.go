package collector

import (
	"errors"
	"fmt"
	"log/slog"
)

func Collect(target string, exporter string) (*string, error) {
	switch exporter {
	case "kafka_exporter":
		return CollectKafkaMetrics(target)
	default:
		return nil, errors.New("exporter not found")
	}
}

func CollectKafkaMetrics(target string) (*string, error) {
	kafkaExporterOption := TargetMap[target].Exporter.KafkaExporter
	if len(kafkaExporterOption.KafkaServer) == 0 {
		slog.Error("kafka: client has run out of available brokers")
		return nil, errors.New("kafka: client has run out of available brokers")
	}
	var kafkaServers string
	for _, v := range kafkaExporterOption.KafkaServer {
		kafkaServers += "--kafka.server=" + v + " "
	}
	if kafkaExporterOption.Port == "" {
		kafkaExporterOption.Port = "9308"
	}
	command := fmt.Sprintf("(.promethues_exporter/kafka_exporter --web.listen-address=:%[2]s %[1]s > /dev/null 2>&1 & sleep 3; curl -s http://localhost:%[2]s/metrics) && pkill kafka_exporter", kafkaServers, kafkaExporterOption.Port)
	t := TargetMap[target]
	return t.SshRoute(&command)
}
