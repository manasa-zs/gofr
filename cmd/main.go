package main

import (
	"gofr.dev/cmd/deploy"
	"gofr.dev/pkg/gofr"
)

func main() {
	app := gofr.NewCMD()

	app.AddHTTPService("deployment", "http://172.0.243.205:8000")

	app.SubCommand("deploy", deploy.Run)

	app.Run()
}
