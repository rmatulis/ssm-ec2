package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/charmbracelet/huh"
)

// InstanceInfo holds combined data for display
type InstanceInfo struct {
	InstanceID   string
	Name         string
	ComputerName string
	Platform     string
}

func main() {
	// 1. Define and parse CLI flags
	// MODIFIED: Set default region to "ap-southeast-2"
	region := flag.String("region", "ap-southeast-2", "The AWS region to target (default: 'ap-southeast-2')")
	profile := flag.String("profile", "default", "The AWS profile to use (default: 'default')")
	flag.Parse()

	// MODIFIED: Removed the check for a blank region, as it now has a default.

	// 2. Check if 'session-manager-plugin' is installed
	if err := checkPluginExists(); err != nil {
		log.Fatalf("FATAL: %v", err)
	}

	log.Printf("Using profile: %s, region: %s\n", *profile, *region)
	ctx := context.Background()

	// 3. Load AWS configuration
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(*region),
		config.WithSharedConfigProfile(*profile),
	)
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	// 4. Get SSM-managed instances
	ssmInstances, err := getManagedInstances(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to get SSM instances: %v", err)
	}
	if len(ssmInstances) == 0 {
		log.Fatal("No SSM-managed instances found (or none are 'Online').")
	}

	// 5. Get EC2 Tags for the instances
	instanceIDs := make([]string, len(ssmInstances))
	ssmInstanceMap := make(map[string]ssmtypes.InstanceInformation)
	for i, inst := range ssmInstances {
		instanceIDs[i] = *inst.InstanceId
		ssmInstanceMap[*inst.InstanceId] = inst
	}

	tags, err := getEC2Tags(ctx, cfg, instanceIDs)
	if err != nil {
		log.Printf("Warning: Could not fetch EC2 'Name' tags: %v", err)
		// Continue without tags
	}

	// 6. Build the list for the selector
	var displayInstances []InstanceInfo
	for _, id := range instanceIDs {
		inst := ssmInstanceMap[id]
		displayInstances = append(displayInstances, InstanceInfo{
			InstanceID:   id,
			Name:         tags[id], // Will be empty string if not found
			ComputerName: aws.ToString(inst.ComputerName),
			Platform:     string(inst.PlatformType),
		})
	}

	// 7. Show the instance selector
	selectedInstanceID, err := selectInstance(displayInstances)
	if err != nil {
		log.Fatalf("Instance selection failed: %v", err)
	}

	log.Printf("Starting SSM session for %s...", selectedInstanceID)

	// 8. Start the SSM session
	if err := startSSMSession(ctx, cfg, selectedInstanceID); err != nil {
		log.Fatalf("SSM session failed: %v", err)
	}

	log.Println("SSM session ended.")
}

// checkPluginExists verifies the 'session-manager-plugin' is in the PATH
func checkPluginExists() error {
	_, err := exec.LookPath("session-manager-plugin")
	if err != nil {
		return fmt.Errorf("'session-manager-plugin' not found in PATH. Please install it first")
	}
	return nil
}

// getManagedInstances fetches all instances managed by SSM and are "Online"
func getManagedInstances(ctx context.Context, cfg aws.Config) ([]ssmtypes.InstanceInformation, error) {
	client := ssm.NewFromConfig(cfg)
	var allInstances []ssmtypes.InstanceInformation

	paginator := ssm.NewDescribeInstanceInformationPaginator(client, &ssm.DescribeInstanceInformationInput{
		InstanceInformationFilterList: []ssmtypes.InstanceInformationFilter{
			{
				Key:      ssmtypes.InstanceInformationFilterKeyPingStatus,
				ValueSet: []string{"Online"},
			},
		},
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe instances: %w", err)
		}
		allInstances = append(allInstances, page.InstanceInformationList...)
	}
	return allInstances, nil
}

// getEC2Tags fetches the 'Name' tag for a list of instance IDs
func getEC2Tags(ctx context.Context, cfg aws.Config, instanceIDs []string) (map[string]string, error) {
	client := ec2.NewFromConfig(cfg)
	tags := make(map[string]string)

	paginator := ec2.NewDescribeInstancesPaginator(client, &ec2.DescribeInstancesInput{
		InstanceIds: instanceIDs,
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to describe instances for tags: %w", err)
		}

		for _, res := range page.Reservations {
			for _, inst := range res.Instances {
				for _, tag := range inst.Tags {
					if aws.ToString(tag.Key) == "Name" {
						tags[aws.ToString(inst.InstanceId)] = aws.ToString(tag.Value)
						break
					}
				}
			}
		}
	}
	return tags, nil
}

// selectInstance shows an interactive menu to pick an instance
func selectInstance(instances []InstanceInfo) (string, error) {
	var options []huh.Option[string]
	for _, inst := range instances {
		// Create a formatted label for the option
		var labelParts []string
		labelParts = append(labelParts, inst.InstanceID)
		if inst.Name != "" {
			labelParts = append(labelParts, fmt.Sprintf("(%s)", inst.Name))
		}
		if inst.ComputerName != "" {
			labelParts = append(labelParts, fmt.Sprintf("- %s", inst.ComputerName))
		}
		labelParts = append(labelParts, fmt.Sprintf("[%s]", inst.Platform))

		label := strings.Join(labelParts, " ")
		options = append(options, huh.NewOption(label, inst.InstanceID))
	}

	var selectedInstanceID string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select an Instance to Connect").
				Options(options...).
				Value(&selectedInstanceID),
		),
	)

	err := form.Run()
	if err != nil {
		return "", err
	}
	return selectedInstanceID, nil
}

// startSSMSession starts the session and hands control to the plugin
func startSSMSession(ctx context.Context, cfg aws.Config, instanceID string) error {
	client := ssm.NewFromConfig(cfg)

	// 1. Call StartSession to get connection details
	resp, err := client.StartSession(ctx, &ssm.StartSessionInput{
		Target: aws.String(instanceID),
	})
	if err != nil {
		return fmt.Errorf("failed to start SSM session: %w", err)
	}

	// 2. Marshal the response to JSON for the plugin
	sessionJSON, err := json.Marshal(resp)
	if err != nil {
		return fmt.Errorf("failed to marshal session response: %w", err)
	}

	// 3. Prepare the command to execute the plugin
	cmd := exec.Command("session-manager-plugin",
		string(sessionJSON),
		cfg.Region,
		"StartSession",
	)

	// 4. Wire up STDIN, STDOUT, and STDERR to the plugin
	// This gives the user interactive control over the shell
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// 5. Run the command. This will block until the user exits the shell.
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("session-manager-plugin failed: %w", err)
	}

	return nil
}
