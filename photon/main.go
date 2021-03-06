// Copyright (c) 2016 VMware, Inc. All Rights Reserved.
//
// This product is licensed to you under the Apache License, Version 2.0 (the "License").
// You may not use this product except in compliance with the License.
//
// This product may include a number of subcomponents with separate copyright notices and
// license terms. Your use of these subcomponents is subject to the terms and conditions
// of the subcomponent's license, as noted in the LICENSE file.

package main

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/vmware/photon-controller-cli/photon/client"
	"github.com/vmware/photon-controller-cli/photon/command"
	"github.com/vmware/photon-controller-cli/photon/utils"
	"os"
)

var commandName = ""
var githash = ""

func main() {
	app := BuildApp()
	err := app.Run(os.Args)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func BuildApp() *cli.App {
	app := cli.NewApp()
	app.Name = commandName
	app.Usage = "Command line interface for Photon Controller"
	app.Version = "Git commit hash: " + githash
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "non-interactive, n",
			Usage: "trigger for non-interactive mode (scripting)",
		},
		cli.StringFlag{
			Name:  "log-file, l",
			Usage: "writes logging information into a logfile at the specified path",
		},
		cli.StringFlag{
			Name:  "output, o",
			Usage: "Select output format",
		},
	}
	app.Commands = []cli.Command{
		command.GetAuthCommand(),
		command.GetSystemCommand(),
		command.GetTargetCommand(),
		command.GetTenantsCommand(),
		command.GetHostsCommand(),
		command.GetDeploymentsCommand(),
		command.GetResourceTicketCommand(),
		command.GetImagesCommand(),
		command.GetTasksCommand(),
		command.GetFlavorsCommand(),
		command.GetProjectsCommand(),
		command.GetDiskCommand(),
		command.GetVMCommand(),
		command.GetNetworksCommand(),
		command.GetClusterCommand(),
		command.GetAvailabilityZonesCommand(),
	}
	app.Before = func(c *cli.Context) error {
		logFile := c.GlobalString("log-file")
		if logFile != "" {
			return client.InitializeLogging(logFile)
		}
		return utils.ValidateArgs(c)
	}
	app.After = func(c *cli.Context) error {
		logFile := c.GlobalString("log-file")
		if logFile != "" {
			return client.CleanupLogging()
		}
		return nil
	}
	return app
}
