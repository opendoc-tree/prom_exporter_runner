package main

import (
	"flag"
	"log"
	"prom_exporter_runner/collector"

	"github.com/gofiber/fiber/v2"
)

var config_file string
var web_listen_address string

func init() {
	flag.StringVar(&config_file, "config.file", "/etc/config.yml", "Configuration file path")
	flag.StringVar(&web_listen_address, "web.listen-address", ":3001", "Address to listen on for web interface")
	flag.Parse()
	collector.LoadConfig(config_file)
}

func main() {
	app := fiber.New()

	app.Get("/encrypt/:param", func(c *fiber.Ctx) error {
		encryptText, _ := collector.Encrypt(c.Params("param"))
		return c.SendString(encryptText)
	})

	app.Get("/metrics", func(c *fiber.Ctx) error {
		targetHost := c.Query("target")
		exporter := c.Query("exporter")
		collect := collector.Collect(targetHost, exporter)
		return c.SendString(collect)
	})

	log.Fatal(app.Listen(web_listen_address))
}
