package main

import (
	"gofr.dev/cmd/deploy"
	"gofr.dev/pkg/gofr"
)

func main() {
	app := gofr.NewCMD()

	app.AddHTTPService("deployment", "localhost:9000")

	app.SubCommand("deploy", deploy.Run)

	app.Run()
}
