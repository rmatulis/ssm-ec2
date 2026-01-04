package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/manifoldco/promptui"
)

// Version information (set by build flags)
var Version = "dev"

type Instance struct {
	ID           string
	Name         string
	PrivateIP    string
	PublicIP     string
	State        string
	InstanceType string
}

type RDSInstance struct {
	Identifier string
	Endpoint   string
	Port       int32
	Engine     string
	Status     string
}

func main() {
	profile := flag.String("profile", "", "AWS profile to use")
	region := flag.String("region", "", "AWS region (optional, uses default region if not specified)")
	mode := flag.String("mode", "", "Mode: 'ec2' for SSM connection or 'rds' for IAM auth token")
	version := flag.Bool("version", false, "Print version information")
	flag.Parse()

	if *version {
		fmt.Printf("ec2-ssm-connector version %s\n", Version)
		return
	}

	ctx := context.Background()

	// Load AWS configuration
	var cfg aws.Config
	var err error

	configOptions := []func(*config.LoadOptions) error{}

	if *profile != "" {
		configOptions = append(configOptions, config.WithSharedConfigProfile(*profile))
	}

	if *region != "" {
		configOptions = append(configOptions, config.WithRegion(*region))
	}

	cfg, err = config.LoadDefaultConfig(ctx, configOptions...)
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	// Determine mode if not specified
	selectedMode := *mode
	if selectedMode == "" {
		selectedMode, err = selectMode()
		if err != nil {
			log.Fatalf("Failed to select mode: %v", err)
		}
	}

	switch selectedMode {
	case "ec2":
		handleEC2Mode(ctx, cfg)
	case "rds":
		handleRDSMode(ctx, cfg)
	default:
		log.Fatalf("Invalid mode: %s. Use 'ec2' or 'rds'", selectedMode)
	}
}

func selectMode() (string, error) {
	modes := []struct {
		Name        string
		Description string
	}{
		{Name: "ec2", Description: "Connect to EC2 instance via SSM"},
		{Name: "rds", Description: "Generate RDS IAM auth token"},
	}

	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "▸ {{ .Description | cyan }}",
		Inactive: "  {{ .Description }}",
		Selected: "✔ Selected: {{ .Description | cyan }}",
	}

	prompt := promptui.Select{
		Label:     "Select operation mode",
		Items:     modes,
		Templates: templates,
	}

	index, _, err := prompt.Run()
	if err != nil {
		return "", err
	}

	return modes[index].Name, nil
}

func handleEC2Mode(ctx context.Context, cfg aws.Config) {
	// List EC2 instances
	instances, err := listInstances(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to list instances: %v", err)
	}

	if len(instances) == 0 {
		fmt.Println("No EC2 instances found")
		return
	}

	// Display instances
	displayInstances(instances)

	// Prompt user to select an instance
	selectedInstance, err := selectInstance(instances)
	if err != nil {
		log.Fatalf("Failed to select instance: %v", err)
	}

	// Connect via SSM
	err = connectToInstance(ctx, cfg, selectedInstance)
	if err != nil {
		log.Fatalf("Failed to connect to instance: %v", err)
	}
}

func handleRDSMode(ctx context.Context, cfg aws.Config) {
	// List RDS instances
	rdsInstances, err := listRDSInstances(ctx, cfg)
	if err != nil {
		log.Fatalf("Failed to list RDS instances: %v", err)
	}

	if len(rdsInstances) == 0 {
		fmt.Println("No RDS instances found")
		return
	}

	// Display RDS instances
	displayRDSInstances(rdsInstances)

	// Prompt user to select an RDS instance
	selectedRDS, err := selectRDSInstance(rdsInstances)
	if err != nil {
		log.Fatalf("Failed to select RDS instance: %v", err)
	}

	// Prompt for username
	username, err := promptForUsername()
	if err != nil {
		log.Fatalf("Failed to get username: %v", err)
	}

	// Generate IAM auth token
	err = generateRDSAuthToken(ctx, cfg, selectedRDS, username)
	if err != nil {
		log.Fatalf("Failed to generate auth token: %v", err)
	}
}

func listInstances(ctx context.Context, cfg aws.Config) ([]Instance, error) {
	ec2Client := ec2.NewFromConfig(cfg)

	// Describe all instances
	result, err := ec2Client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to describe instances: %w", err)
	}

	var instances []Instance

	for _, reservation := range result.Reservations {
		for _, instance := range reservation.Instances {
			// Skip terminated instances
			if instance.State.Name == types.InstanceStateNameTerminated {
				continue
			}

			inst := Instance{
				ID:           aws.ToString(instance.InstanceId),
				InstanceType: string(instance.InstanceType),
				State:        string(instance.State.Name),
			}

			// Get instance name from tags
			for _, tag := range instance.Tags {
				if aws.ToString(tag.Key) == "Name" {
					inst.Name = aws.ToString(tag.Value)
					break
				}
			}

			// Get IP addresses
			if instance.PrivateIpAddress != nil {
				inst.PrivateIP = aws.ToString(instance.PrivateIpAddress)
			}
			if instance.PublicIpAddress != nil {
				inst.PublicIP = aws.ToString(instance.PublicIpAddress)
			}

			instances = append(instances, inst)
		}
	}

	return instances, nil
}

func displayInstances(instances []Instance) {
	fmt.Println("\nAvailable EC2 Instances:")
	fmt.Println(strings.Repeat("=", 120))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "INDEX\tINSTANCE ID\tNAME\tSTATE\tTYPE\tPRIVATE IP\tPUBLIC IP")
	fmt.Fprintln(w, strings.Repeat("-", 5)+"\t"+strings.Repeat("-", 19)+"\t"+strings.Repeat("-", 30)+"\t"+
		strings.Repeat("-", 10)+"\t"+strings.Repeat("-", 12)+"\t"+strings.Repeat("-", 15)+"\t"+strings.Repeat("-", 15))

	for i, inst := range instances {
		name := inst.Name
		if name == "" {
			name = "-"
		}
		publicIP := inst.PublicIP
		if publicIP == "" {
			publicIP = "-"
		}

		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\t%s\n",
			i+1, inst.ID, name, inst.State, inst.InstanceType, inst.PrivateIP, publicIP)
	}
	w.Flush()
	fmt.Println(strings.Repeat("=", 120))
}

func selectInstance(instances []Instance) (Instance, error) {
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "▸ {{ .Name | cyan }} ({{ .ID | yellow }}) - {{ .State | green }}",
		Inactive: "  {{ .Name }} ({{ .ID }}) - {{ .State }}",
		Selected: "✔ Selected: {{ .Name | cyan }} ({{ .ID | yellow }})",
	}

	prompt := promptui.Select{
		Label:     "Select an EC2 instance to connect",
		Items:     instances,
		Templates: templates,
		Size:      10,
	}

	index, _, err := prompt.Run()
	if err != nil {
		return Instance{}, err
	}

	return instances[index], nil
}

func connectToInstance(ctx context.Context, cfg aws.Config, instance Instance) error {
	// Check if instance is running
	if instance.State != string(types.InstanceStateNameRunning) {
		return fmt.Errorf("instance %s is not in running state (current state: %s)", instance.ID, instance.State)
	}

	// Check if session-manager-plugin is installed
	if _, err := exec.LookPath("session-manager-plugin"); err != nil {
		return fmt.Errorf("session-manager-plugin not found. Please install it from: https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html")
	}

	ssmClient := ssm.NewFromConfig(cfg)

	// Start SSM session
	fmt.Printf("\nStarting SSM session to %s (%s)...\n", instance.Name, instance.ID)

	startSessionInput := &ssm.StartSessionInput{
		Target: aws.String(instance.ID),
	}

	result, err := ssmClient.StartSession(ctx, startSessionInput)
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}

	// Build the session-manager-plugin command
	sessionID := aws.ToString(result.SessionId)
	streamURL := aws.ToString(result.StreamUrl)
	tokenValue := aws.ToString(result.TokenValue)

	// Get the region from config
	region := cfg.Region

	// Prepare the session data JSON
	sessionData := fmt.Sprintf(`{"SessionId":"%s","StreamUrl":"%s","TokenValue":"%s"}`, sessionID, streamURL, tokenValue)

	// Execute session-manager-plugin
	cmd := exec.Command(
		"session-manager-plugin",
		sessionData,
		region,
		"StartSession",
	)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Println("Connected! Type 'exit' to close the session.")

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("session-manager-plugin error: %w", err)
	}

	fmt.Println("\nSession ended.")
	return nil
}

// RDS-related functions

func listRDSInstances(ctx context.Context, cfg aws.Config) ([]RDSInstance, error) {
	rdsClient := rds.NewFromConfig(cfg)

	// Describe all RDS instances
	result, err := rdsClient.DescribeDBInstances(ctx, &rds.DescribeDBInstancesInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to describe RDS instances: %w", err)
	}

	var instances []RDSInstance

	for _, dbInstance := range result.DBInstances {
		engine := aws.ToString(dbInstance.Engine)

		// Skip Oracle databases as they don't support IAM authentication
		if strings.Contains(strings.ToLower(engine), "oracle") {
			continue
		}

		inst := RDSInstance{
			Identifier: aws.ToString(dbInstance.DBInstanceIdentifier),
			Engine:     engine,
			Status:     aws.ToString(dbInstance.DBInstanceStatus),
		}

		if dbInstance.Endpoint != nil {
			inst.Endpoint = aws.ToString(dbInstance.Endpoint.Address)
			if dbInstance.Endpoint.Port != nil {
				inst.Port = *dbInstance.Endpoint.Port
			}
		}

		instances = append(instances, inst)
	}

	return instances, nil
}

func displayRDSInstances(instances []RDSInstance) {
	fmt.Println("\nAvailable RDS Instances:")
	fmt.Println(strings.Repeat("=", 120))

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "INDEX\tIDENTIFIER\tENGINE\tSTATUS\tENDPOINT\tPORT")
	fmt.Fprintln(w, strings.Repeat("-", 5)+"\t"+strings.Repeat("-", 40)+"\t"+
		strings.Repeat("-", 15)+"\t"+strings.Repeat("-", 15)+"\t"+strings.Repeat("-", 50)+"\t"+strings.Repeat("-", 6))

	for i, inst := range instances {
		endpoint := inst.Endpoint
		if endpoint == "" {
			endpoint = "-"
		}
		port := fmt.Sprintf("%d", inst.Port)
		if inst.Port == 0 {
			port = "-"
		}

		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
			i+1, inst.Identifier, inst.Engine, inst.Status, endpoint, port)
	}
	w.Flush()
	fmt.Println(strings.Repeat("=", 120))
}

func selectRDSInstance(instances []RDSInstance) (RDSInstance, error) {
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}",
		Active:   "▸ {{ .Identifier | cyan }} ({{ .Engine | yellow }}) - {{ .Status | green }}",
		Inactive: "  {{ .Identifier }} ({{ .Engine }}) - {{ .Status }}",
		Selected: "✔ Selected: {{ .Identifier | cyan }} ({{ .Engine | yellow }})",
	}

	prompt := promptui.Select{
		Label:     "Select an RDS instance",
		Items:     instances,
		Templates: templates,
		Size:      10,
	}

	index, _, err := prompt.Run()
	if err != nil {
		return RDSInstance{}, err
	}

	return instances[index], nil
}

func promptForUsername() (string, error) {
	validate := func(input string) error {
		if strings.TrimSpace(input) == "" {
			return fmt.Errorf("username cannot be empty")
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    "Enter database username",
		Validate: validate,
	}

	username, err := prompt.Run()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(username), nil
}

func generateRDSAuthToken(ctx context.Context, cfg aws.Config, instance RDSInstance, username string) error {
	if instance.Status != "available" {
		fmt.Printf("Warning: RDS instance %s is not in 'available' state (current state: %s)\n", instance.Identifier, instance.Status)
	}

	if instance.Endpoint == "" {
		return fmt.Errorf("RDS instance %s does not have an endpoint", instance.Identifier)
	}

	// Build the endpoint address with port
	endpoint := fmt.Sprintf("%s:%d", instance.Endpoint, instance.Port)

	fmt.Printf("\nGenerating IAM authentication token for:\n")
	fmt.Printf("  Instance: %s\n", instance.Identifier)
	fmt.Printf("  Endpoint: %s\n", endpoint)
	fmt.Printf("  Username: %s\n", username)
	fmt.Printf("  Region:   %s\n", cfg.Region)
	fmt.Println()

	// Generate the auth token (valid for 15 minutes)
	authToken, err := auth.BuildAuthToken(ctx, endpoint, cfg.Region, username, cfg.Credentials)
	if err != nil {
		return fmt.Errorf("failed to generate auth token: %w", err)
	}

	fmt.Println("IAM Authentication Token (valid for 15 minutes):")
	fmt.Println(strings.Repeat("=", 120))
	fmt.Println(authToken)
	fmt.Println(strings.Repeat("=", 120))
	fmt.Println()

	// Display connection examples
	fmt.Println("Connection Examples:")
	fmt.Println()

	// MySQL/MariaDB
	if strings.Contains(strings.ToLower(instance.Engine), "mysql") || strings.Contains(strings.ToLower(instance.Engine), "mariadb") {
		fmt.Println("MySQL/MariaDB:")
		fmt.Printf("  mysql -h %s -P %d -u %s --password='%s' --enable-cleartext-plugin --ssl-mode=REQUIRED\n",
			instance.Endpoint, instance.Port, username, authToken)
		fmt.Println()
	}

	// PostgreSQL
	if strings.Contains(strings.ToLower(instance.Engine), "postgres") {
		fmt.Println("PostgreSQL:")
		fmt.Printf("  psql \"host=%s port=%d dbname=your_database user=%s password=%s sslmode=require\"\n",
			instance.Endpoint, instance.Port, username, authToken)
		fmt.Println()
		fmt.Println("Or using environment variable:")
		fmt.Printf("  export PGPASSWORD='%s'\n", authToken)
		fmt.Printf("  psql -h %s -p %d -U %s -d your_database\n", instance.Endpoint, instance.Port, username)
		fmt.Println()
	}

	fmt.Println("Notes:")
	fmt.Println("  - Token is valid for 15 minutes from generation time")
	fmt.Println("  - IAM database authentication must be enabled on the RDS instance")
	fmt.Println("  - The database user must be configured to use IAM authentication")
	fmt.Println("  - SSL/TLS connection is required")
	fmt.Printf("  - Generated at: %s\n", time.Now().Format(time.RFC3339))

	return nil
}
