package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/rds/auth"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/spf13/cobra"
)

// Version information (set by build flags)
var Version = "dev"

// Global flags
var (
	profile string
	region  string
)

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

// SessionData represents the data structure for SSM session manager plugin
type SessionData struct {
	SessionId  string `json:"SessionId"`
	StreamUrl  string `json:"StreamUrl"`
	TokenValue string `json:"TokenValue"`
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "aws-go-tools",
		Short: "AWS tools for EC2 SSM connections and RDS IAM authentication",
		Long:  `A CLI tool to connect to EC2 instances via SSM and generate RDS IAM authentication tokens.`,
	}

	// Version command
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("aws-go-tools version %s\n", Version)
		},
	}

	// EC2 command
	ec2Cmd := &cobra.Command{
		Use:   "ec2",
		Short: "Connect to EC2 instance via SSM",
		Long:  `List EC2 instances and connect to the selected instance using AWS Systems Manager Session Manager.`,
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			cfg := loadAWSConfig(ctx)
			handleEC2Mode(ctx, cfg)
		},
	}

	// RDS command
	rdsCmd := &cobra.Command{
		Use:   "rds",
		Short: "Generate RDS IAM authentication token",
		Long:  `List RDS instances and generate an IAM authentication token for the selected instance.`,
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			cfg := loadAWSConfig(ctx)
			handleRDSMode(ctx, cfg)
		},
	}

	// Add persistent flags
	rootCmd.PersistentFlags().StringVarP(&profile, "profile", "p", "", "AWS profile to use")
	rootCmd.PersistentFlags().StringVarP(&region, "region", "r", "", "AWS region (optional, uses default region if not specified)")

	// Add commands
	rootCmd.AddCommand(versionCmd, ec2Cmd, rdsCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func loadAWSConfig(ctx context.Context) aws.Config {
	configOptions := []func(*config.LoadOptions) error{}

	if profile != "" {
		configOptions = append(configOptions, config.WithSharedConfigProfile(profile))
	}

	if region != "" {
		configOptions = append(configOptions, config.WithRegion(region))
	}

	cfg, err := config.LoadDefaultConfig(ctx, configOptions...)
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	return cfg
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
	fmt.Fprintln(w, "INSTANCE ID\tNAME\tSTATE\tTYPE\tPRIVATE IP\tPUBLIC IP")
	fmt.Fprintln(w, strings.Repeat("-", 19)+"\t"+strings.Repeat("-", 30)+"\t"+
		strings.Repeat("-", 10)+"\t"+strings.Repeat("-", 12)+"\t"+strings.Repeat("-", 15)+"\t"+strings.Repeat("-", 15))

	for _, inst := range instances {
		name := inst.Name
		if name == "" {
			name = "-"
		}
		publicIP := inst.PublicIP
		if publicIP == "" {
			publicIP = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			inst.ID, name, inst.State, inst.InstanceType, inst.PrivateIP, publicIP)
	}
	w.Flush()
	fmt.Println(strings.Repeat("=", 120))
}

func selectInstance(instances []Instance) (Instance, error) {
	var options []string
	for _, inst := range instances {
		options = append(options, fmt.Sprintf("%s (%s) - %s", inst.Name, inst.ID, inst.State))
	}

	var selected string
	prompt := &survey.Select{
		Message: "Select an EC2 instance to connect:",
		Options: options,
		PageSize: 10,
	}

	err := survey.AskOne(prompt, &selected)
	if err != nil {
		return Instance{}, err
	}

	// Find the selected instance by matching the formatted string
	for i, opt := range options {
		if opt == selected {
			return instances[i], nil
		}
	}

	return Instance{}, fmt.Errorf("selected instance not found")
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

	// Prepare the session data JSON using proper marshaling
	sessionDataStruct := SessionData{
		SessionId:  sessionID,
		StreamUrl:  streamURL,
		TokenValue: tokenValue,
	}

	sessionDataBytes, err := json.Marshal(sessionDataStruct)
	if err != nil {
		return fmt.Errorf("failed to marshal session data: %w", err)
	}
	sessionData := string(sessionDataBytes)

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
	fmt.Fprintln(w, "IDENTIFIER\tENGINE\tSTATUS\tENDPOINT\tPORT")
	fmt.Fprintln(w, strings.Repeat("-", 40)+"\t"+
		strings.Repeat("-", 15)+"\t"+strings.Repeat("-", 15)+"\t"+strings.Repeat("-", 50)+"\t"+strings.Repeat("-", 6))

	for _, inst := range instances {
		endpoint := inst.Endpoint
		if endpoint == "" {
			endpoint = "-"
		}
		port := fmt.Sprintf("%d", inst.Port)
		if inst.Port == 0 {
			port = "-"
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			inst.Identifier, inst.Engine, inst.Status, endpoint, port)
	}
	w.Flush()
	fmt.Println(strings.Repeat("=", 120))
}

func selectRDSInstance(instances []RDSInstance) (RDSInstance, error) {
	var options []string
	for _, inst := range instances {
		options = append(options, fmt.Sprintf("%s (%s) - %s", inst.Identifier, inst.Engine, inst.Status))
	}

	var selected string
	prompt := &survey.Select{
		Message: "Select an RDS instance:",
		Options: options,
		PageSize: 10,
	}

	err := survey.AskOne(prompt, &selected)
	if err != nil {
		return RDSInstance{}, err
	}

	// Find the selected instance by matching the formatted string
	for i, opt := range options {
		if opt == selected {
			return instances[i], nil
		}
	}

	return RDSInstance{}, fmt.Errorf("selected RDS instance not found")
}

func promptForUsername() (string, error) {
	var username string
	prompt := &survey.Input{
		Message: "Enter database username:",
	}

	err := survey.AskOne(prompt, &username, survey.WithValidator(survey.Required))
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
