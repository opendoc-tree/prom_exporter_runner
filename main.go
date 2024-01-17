package main

import (
	"flag"
	"log"
	"os"
	"prom_exporter_runner/collector"

	"github.com/gofiber/fiber/v2"
)

var config_file string
var web_listen_address string

func init() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "secret":
			collector.CreateSecret()
		case "encrypt":
			collector.GetEncryptTxt()
		}
	}
	flag.StringVar(&config_file, "config.file", "/etc/config.yml", "Configuration file path")
	flag.StringVar(&web_listen_address, "web.listen-address", ":3001", "Address to listen on for web interface")
	flag.Parse()
	collector.LoadSecretKey()
	collector.LoadConfig(config_file)
}

func main() {
	app := fiber.New()

	app.Get("/metrics", func(c *fiber.Ctx) error {
		targetHost := c.Query("target")
		exporter := c.Query("exporter")
		collect := collector.Collect(targetHost, exporter)
		return c.SendString(collect)
	})

	log.Fatal(app.Listen(web_listen_address))
}
