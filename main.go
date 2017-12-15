package main

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	"github.com/zionwu/catalog-images-synchronizer/config"
	"github.com/zionwu/catalog-images-synchronizer/sync"
)

var VERSION = "v0.0.1"

func main() {
	app := cli.NewApp()
	app.Name = "catalog-images-synchronizer"
	app.Author = "zionwu"
	app.Version = VERSION
	app.Usage = "Synchronize images of catalog from docker hub to harbor."
	app.Action = run
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:   "debug, d",
			Usage:  "Debug logging",
			EnvVar: "DEBUG",
		},
		cli.StringFlag{
			Name:   "harbor_username",
			Usage:  "Harbor Username",
			EnvVar: "HARBOR_USERNAME",
		},
		cli.StringFlag{
			Name:   "harbor_password",
			Usage:  "Harbor Password",
			EnvVar: "HARBOR_PASSWORD",
		},
		cli.StringFlag{
			Name:   "harbor_address",
			Usage:  "Harbor address",
			EnvVar: "HARBOR_ADDRESS",
		},
		cli.StringFlag{
			Name:   "catalog_url",
			Usage:  "Catalog Git Repo URL",
			EnvVar: "CATALOG_URL",
		},
		cli.StringFlag{
			Name:   "catalog_branch",
			Usage:  "Catalog Git branch",
			EnvVar: "CATALOG_BRANCH",
		},
	}

	app.Run(os.Args)
}

func run(c *cli.Context) error {

	if c.Bool("debug") {
		logrus.SetLevel(logrus.DebugLevel)
	}

	if err := config.Init(c); err != nil {
		logrus.Errorf("Error initialize config: %v", err)

	}
	if err := sync.NewImageSynchronize().Run(); err != nil {
		logrus.Errorf("Error occurred while synchronizing the images: %v", err)
	}

	return nil
}
