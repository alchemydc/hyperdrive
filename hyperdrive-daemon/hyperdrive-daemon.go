package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/nodeset-org/hyperdrive/hyperdrive-daemon/common"
	"github.com/nodeset-org/hyperdrive/hyperdrive-daemon/server"
	"github.com/nodeset-org/hyperdrive/hyperdrive-daemon/tasks"
	"github.com/nodeset-org/hyperdrive/shared"
	"github.com/nodeset-org/hyperdrive/shared/config"
	"github.com/urfave/cli/v2"
)

// Run
func main() {
	// Add logo and attribution to application help template
	attribution := "ATTRIBUTION:\n   Adapted from the Rocket Pool Smart Node (https://github.com/rocketpool/smartnode) with love."
	cli.AppHelpTemplate = fmt.Sprintf("\n%s\n\n%s\n%s\n", shared.Logo, cli.AppHelpTemplate, attribution)
	cli.CommandHelpTemplate = fmt.Sprintf("%s\n%s\n", cli.CommandHelpTemplate, attribution)
	cli.SubcommandHelpTemplate = fmt.Sprintf("%s\n%s\n", cli.SubcommandHelpTemplate, attribution)

	// Initialise application
	app := cli.NewApp()

	// Set application info
	app.Name = "hyperdrive-daemon"
	app.Usage = "Hyperdrive Daemon for NodeSet Node Operator Management"
	app.Version = shared.HyperdriveVersion
	app.Authors = []*cli.Author{
		{
			Name:  "Joe Clapis",
			Email: "joe@nodeset.io",
		},
	}
	app.Copyright = "(C) 2024 NodeSet LLC"

	userDirFlag := &cli.StringFlag{
		Name:     "user-dir",
		Aliases:  []string{"u"},
		Usage:    "The path of the user data directory, which contains the configuration file to load and all of the user's runtime data",
		Required: true,
	}

	app.Flags = []cli.Flag{
		userDirFlag,
	}
	app.Action = func(c *cli.Context) error {
		// Get the config file
		userDir := c.String(userDirFlag.Name)
		cfgPath := filepath.Join(userDir, config.ConfigFilename)
		_, err := os.Stat(cfgPath)
		if errors.Is(err, fs.ErrNotExist) {
			fmt.Printf("Configuration file not found at [%s].", cfgPath)
			os.Exit(1)
		}

		// Wait group to handle graceful stopping
		stopWg := new(sync.WaitGroup)

		// Create the service provider
		sp, err := common.NewServiceProvider(userDir)
		if err != nil {
			return fmt.Errorf("error creating service provider: %w", err)
		}

		// Create the data dir
		dataDir := sp.GetConfig().UserDataPath.Value
		err = os.MkdirAll(dataDir, 0700)
		if err != nil {
			return fmt.Errorf("error creating user data directory [%s]: %w", dataDir, err)
		}

		// Create the server manager
		serverMgr, err := server.NewServerManager(sp, cfgPath, stopWg)
		if err != nil {
			return fmt.Errorf("error creating server manager: %w", err)
		}

		// Start the task loop
		taskLoop := tasks.NewTaskLoop(sp, stopWg)
		err = taskLoop.Run()
		if err != nil {
			return fmt.Errorf("error starting task loop: %w", err)
		}

		// TODO: Metrics manager

		// Handle process closures
		termListener := make(chan os.Signal, 1)
		signal.Notify(termListener, os.Interrupt, syscall.SIGTERM)
		go func() {
			<-termListener
			fmt.Println("Shutting down daemon...")
			serverMgr.Stop()
			taskLoop.Stop()
		}()

		// Run the daemon until closed
		socketPath := filepath.Join(userDir, config.HyperdriveSocketFilename)
		fmt.Printf("Started daemon on %s.\n", socketPath)
		stopWg.Wait()
		fmt.Println("Daemon stopped.")
		return nil
	}

	// Run application
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
