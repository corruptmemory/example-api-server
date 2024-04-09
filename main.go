package main

import (
	"errors"
	"fmt"
	"log"
	"net/netip"

	"example-api-server/app"
	"example-api-server/webapp"

	"github.com/jessevdk/go-flags"
	hd "github.com/mitchellh/go-homedir"
)

type Args struct {
	ConfigFile string `short:"c" long:"config-file" description:"Data exporter config file" default:"./example-api-server.toml"`
	Address    string `short:"a" long:"address" description:"The address to listen on for HTTP requests" default:"0.0.0.0"`
	Port       int    `short:"p" long:"port" description:"The port to listen on for HTTP requests"`

	config *Config
}

func (a *Args) validate() (err error) {
	if a.ConfigFile == "" {
		return errors.New("error: config-file is required")
	}
	cf, err := hd.Expand(a.ConfigFile)
	if err != nil {
		return fmt.Errorf("error: could not expand config-file path[%s]: %v", a.ConfigFile, err)
	}
	a.ConfigFile = cf
	a.config, err = loadConfig(a.ConfigFile)
	if err != nil {
		return fmt.Errorf("error: error occurred loading the config file[%s]: %v", a.ConfigFile, err)
	}
	if a.Port == 0 {
		a.Port = a.config.Port
	}
	if a.Port == 0 {
		return errors.New("error: port is required")
	}
	if a.Port < 0 || a.Port > 65535 {
		return errors.New("error: port must be between 0 and 65535")
	}

	if a.Address == "" {
		a.Address = a.config.Address
	}
	_, err = netip.ParseAddr(a.Address)
	if err != nil {
		return fmt.Errorf("error: invalid address[%s]: %v", a.Address, err)
	}
	return nil
}

func main() {
	var args Args
	parser := flags.NewParser(&args, flags.Default)
	_, err := parser.Parse()
	if err != nil {
		return
	}
	err = args.validate()
	if err != nil {
		log.Fatalf("%v\n", err)
		return
	}

	ap := app.NewApp(100)
	wapp := webapp.NewWebApp(ap)
	srv := webapp.NewServerWithAddress(args.Address, uint(args.Port), wapp)
	srv.Start()
	srv.Wait()
}
