package config

import "github.com/urfave/cli"
import "fmt"

type Config struct {
	HarborUserName string
	HarborPassword string
	HarborAddress  string
	CatalogUrl     string
	CatalogBranch  string
}

var config Config

func Init(c *cli.Context) error {
	config.HarborPassword = c.String("harbor_password")
	if config.HarborPassword == "" {
		return fmt.Errorf("Missing harbor password")
	}
	config.HarborUserName = c.String("harbor_username")
	if config.HarborUserName == "" {
		return fmt.Errorf("Missing harbor username")
	}
	config.HarborAddress = c.String("harbor_address")
	if config.HarborAddress == "" {
		return fmt.Errorf("Missing harbor address")
	}
	config.CatalogBranch = c.String("catalog_branch")
	config.CatalogUrl = c.String("catalog_url")
	if config.CatalogUrl == "" {
		return fmt.Errorf("Missing catalog url")
	}

	return nil

}

func GetConfig() Config {
	return config
}
