package service

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/nodeset-org/hyperdrive/hyperdrive-cli/client"
	"github.com/nodeset-org/hyperdrive/hyperdrive-cli/utils"
	"github.com/nodeset-org/hyperdrive/hyperdrive-cli/utils/terminal"
	"github.com/nodeset-org/hyperdrive/shared"
	"github.com/urfave/cli/v2"
)

// Start the Hyperdrive service
func startService(c *cli.Context, ignoreConfigSuggestion bool) error {
	// Get RP client
	hd := client.NewClientFromCtx(c)

	// Update the Prometheus template with the assigned ports
	cfg, isNew, err := hd.LoadConfig()
	if err != nil {
		return fmt.Errorf("Error loading user settings: %w", err)
	}

	if isNew {
		return fmt.Errorf("No configuration detected. Please run `hyperdrive service config` to set up Hyperdrive before running it.")
	}

	// Check if this is a new install
	oldVersion := strings.TrimPrefix(cfg.Version, "v")
	currentVersion := strings.TrimPrefix(shared.HyperdriveVersion, "v")
	isUpdate := oldVersion != currentVersion
	if isUpdate && !ignoreConfigSuggestion {
		if c.Bool(utils.YesFlag.Name) || utils.Confirm("Hyperdrive upgrade detected - starting will overwrite certain settings with the latest defaults (such as container versions).\nYou may want to run `hyperdrive service config` first to see what's changed.\n\nWould you like to continue starting the service?") {
			cfg.UpdateDefaults()
			hd.SaveConfig(cfg)
			fmt.Printf("%sUpdated settings successfully.%s\n", terminal.ColorGreen, terminal.ColorReset)
		} else {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Update the Prometheus and Grafana config templates with the assigned ports
	metricsEnabled := cfg.Metrics.EnableMetrics.Value
	if metricsEnabled {
		err := hd.UpdatePrometheusConfiguration(cfg)
		if err != nil {
			return err
		}
		err = hd.UpdateGrafanaDatabaseConfiguration(cfg)
		if err != nil {
			return err
		}
	}

	// Validate the config
	errors := cfg.Validate()
	if len(errors) > 0 {
		fmt.Printf("%sYour configuration encountered errors. You must correct the following in order to start Hyperdrive:\n\n", terminal.ColorRed)
		for _, err := range errors {
			fmt.Printf("%s\n\n", err)
		}
		fmt.Println(terminal.ColorReset)
		return nil
	}

	// Start service
	err = hd.StartService(getComposeFiles(c))
	if err != nil {
		return err
	}
	return nil
}

// Extract the image name from a Docker image string
func getDockerImageName(imageString string) (string, error) {
	// Return the empty string if the validator didn't exist (probably because this is the first time starting it up)
	if imageString == "" {
		return "", nil
	}

	reg := regexp.MustCompile(dockerImageRegex)
	matches := reg.FindStringSubmatch(imageString)
	if matches == nil {
		return "", fmt.Errorf("Couldn't parse the Docker image string [%s]", imageString)
	}
	imageIndex := reg.SubexpIndex("image")
	if imageIndex == -1 {
		return "", fmt.Errorf("Image name not found in Docker image [%s]", imageString)
	}

	imageName := matches[imageIndex]
	return imageName, nil
}
