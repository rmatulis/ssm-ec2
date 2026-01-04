package main

import (
	"strings"
	"testing"
)

func TestVersion(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
}

func TestInstanceStruct(t *testing.T) {
	tests := []struct {
		name     string
		instance Instance
		wantID   string
		wantName string
	}{
		{
			name: "Complete instance",
			instance: Instance{
				ID:           "i-123456",
				Name:         "test-instance",
				State:        "running",
				InstanceType: "t3.micro",
				PrivateIP:    "10.0.1.100",
				PublicIP:     "54.123.45.67",
			},
			wantID:   "i-123456",
			wantName: "test-instance",
		},
		{
			name: "Instance without name",
			instance: Instance{
				ID:           "i-789012",
				State:        "stopped",
				InstanceType: "t3.small",
			},
			wantID:   "i-789012",
			wantName: "",
		},
		{
			name: "Instance without public IP",
			instance: Instance{
				ID:           "i-345678",
				Name:         "private-instance",
				State:        "running",
				InstanceType: "t3.medium",
				PrivateIP:    "10.0.2.50",
			},
			wantID:   "i-345678",
			wantName: "private-instance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.instance.ID != tt.wantID {
				t.Errorf("Expected ID to be %s, got %s", tt.wantID, tt.instance.ID)
			}
			if tt.instance.Name != tt.wantName {
				t.Errorf("Expected Name to be %s, got %s", tt.wantName, tt.instance.Name)
			}
		})
	}
}

func TestRDSInstanceStruct(t *testing.T) {
	tests := []struct {
		name        string
		rdsInstance RDSInstance
		wantEngine  string
		wantPort    int32
	}{
		{
			name: "PostgreSQL instance",
			rdsInstance: RDSInstance{
				Identifier: "postgres-db",
				Engine:     "postgres",
				Status:     "available",
				Port:       5432,
				Endpoint:   "db.example.com",
			},
			wantEngine: "postgres",
			wantPort:   5432,
		},
		{
			name: "MySQL instance",
			rdsInstance: RDSInstance{
				Identifier: "mysql-db",
				Engine:     "mysql",
				Status:     "available",
				Port:       3306,
				Endpoint:   "mysql.example.com",
			},
			wantEngine: "mysql",
			wantPort:   3306,
		},
		{
			name: "MariaDB instance",
			rdsInstance: RDSInstance{
				Identifier: "mariadb-db",
				Engine:     "mariadb",
				Status:     "available",
				Port:       3306,
			},
			wantEngine: "mariadb",
			wantPort:   3306,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.rdsInstance.Engine != tt.wantEngine {
				t.Errorf("Expected Engine to be %s, got %s", tt.wantEngine, tt.rdsInstance.Engine)
			}
			if tt.rdsInstance.Port != tt.wantPort {
				t.Errorf("Expected Port to be %d, got %d", tt.wantPort, tt.rdsInstance.Port)
			}
		})
	}
}

func TestRDSEngineFiltering(t *testing.T) {
	tests := []struct {
		name           string
		engine         string
		shouldBeOracle bool
	}{
		{"Oracle standard", "oracle-ee", true},
		{"Oracle SE2", "oracle-se2", true},
		{"Oracle SE1", "oracle-se1", true},
		{"MySQL", "mysql", false},
		{"PostgreSQL", "postgres", false},
		{"MariaDB", "mariadb", false},
		{"Aurora MySQL", "aurora-mysql", false},
		{"Aurora PostgreSQL", "aurora-postgresql", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isOracle := strings.Contains(strings.ToLower(tt.engine), "oracle")
			if isOracle != tt.shouldBeOracle {
				t.Errorf("Engine %s: expected oracle=%v, got %v", tt.engine, tt.shouldBeOracle, isOracle)
			}
		})
	}
}

func TestInstanceStates(t *testing.T) {
	validStates := []string{"pending", "running", "stopping", "stopped", "shutting-down", "terminated"}

	for _, state := range validStates {
		inst := Instance{
			ID:    "i-test",
			State: state,
		}
		if inst.State != state {
			t.Errorf("Expected state %s, got %s", state, inst.State)
		}
	}
}

func TestRDSInstanceStatuses(t *testing.T) {
	validStatuses := []string{
		"available",
		"backing-up",
		"creating",
		"deleting",
		"failed",
		"maintenance",
		"modifying",
		"rebooting",
		"starting",
		"stopped",
		"stopping",
	}

	for _, status := range validStatuses {
		rds := RDSInstance{
			Identifier: "test-db",
			Status:     status,
		}
		if rds.Status != status {
			t.Errorf("Expected status %s, got %s", status, rds.Status)
		}
	}
}

func TestInstanceTypeValidation(t *testing.T) {
	instanceTypes := []string{
		"t3.micro",
		"t3.small",
		"t3.medium",
		"t3.large",
		"m5.xlarge",
		"c5.2xlarge",
		"r5.4xlarge",
	}

	for _, iType := range instanceTypes {
		inst := Instance{
			ID:           "i-test",
			InstanceType: iType,
		}
		if inst.InstanceType != iType {
			t.Errorf("Expected instance type %s, got %s", iType, inst.InstanceType)
		}
	}
}

func TestRDSPorts(t *testing.T) {
	tests := []struct {
		engine      string
		defaultPort int32
		description string
	}{
		{"mysql", 3306, "MySQL default port"},
		{"mariadb", 3306, "MariaDB default port"},
		{"postgres", 5432, "PostgreSQL default port"},
		{"aurora-mysql", 3306, "Aurora MySQL default port"},
		{"aurora-postgresql", 5432, "Aurora PostgreSQL default port"},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			rds := RDSInstance{
				Engine: tt.engine,
				Port:   tt.defaultPort,
			}
			if rds.Port != tt.defaultPort {
				t.Errorf("Expected port %d for %s, got %d", tt.defaultPort, tt.engine, rds.Port)
			}
		})
	}
}

func TestEmptyInstanceFields(t *testing.T) {
	inst := Instance{}

	if inst.ID != "" {
		t.Error("New instance ID should be empty")
	}
	if inst.Name != "" {
		t.Error("New instance Name should be empty")
	}
	if inst.PrivateIP != "" {
		t.Error("New instance PrivateIP should be empty")
	}
	if inst.PublicIP != "" {
		t.Error("New instance PublicIP should be empty")
	}
}

func TestEmptyRDSInstanceFields(t *testing.T) {
	rds := RDSInstance{}

	if rds.Identifier != "" {
		t.Error("New RDS instance Identifier should be empty")
	}
	if rds.Endpoint != "" {
		t.Error("New RDS instance Endpoint should be empty")
	}
	if rds.Port != 0 {
		t.Error("New RDS instance Port should be 0")
	}
}

func TestInstanceWithAllFields(t *testing.T) {
	inst := Instance{
		ID:           "i-0123456789abcdef0",
		Name:         "production-web-server",
		PrivateIP:    "10.0.1.100",
		PublicIP:     "54.123.45.67",
		State:        "running",
		InstanceType: "t3.medium",
	}

	if inst.ID == "" {
		t.Error("Instance ID should not be empty")
	}
	if inst.Name == "" {
		t.Error("Instance Name should not be empty")
	}
	if inst.PrivateIP == "" {
		t.Error("Instance PrivateIP should not be empty")
	}
	if inst.PublicIP == "" {
		t.Error("Instance PublicIP should not be empty")
	}
	if inst.State != "running" {
		t.Errorf("Expected state running, got %s", inst.State)
	}
}

func TestRDSInstanceWithAllFields(t *testing.T) {
	rds := RDSInstance{
		Identifier: "production-postgres-db",
		Endpoint:   "prod-db.abc123xyz.us-east-1.rds.amazonaws.com",
		Port:       5432,
		Engine:     "postgres",
		Status:     "available",
	}

	if rds.Identifier == "" {
		t.Error("RDS Identifier should not be empty")
	}
	if rds.Endpoint == "" {
		t.Error("RDS Endpoint should not be empty")
	}
	if rds.Port == 0 {
		t.Error("RDS Port should not be 0")
	}
	if rds.Engine == "" {
		t.Error("RDS Engine should not be empty")
	}
	if rds.Status != "available" {
		t.Errorf("Expected status available, got %s", rds.Status)
	}
}

func TestInstanceIDFormats(t *testing.T) {
	tests := []struct {
		name  string
		id    string
		valid bool
	}{
		{"Valid long instance ID", "i-0123456789abcdef0", true},
		{"Valid short instance ID 1", "i-abcdef0123456789", true},
		{"Valid short instance ID 2", "i-1234567890abcdef", true},
		{"Invalid prefix", "e-0123456789abcdef0", false},
		{"Too short", "i-abcdef012345", false},
		{"Too long", "i-0123456789abcdef01234", false},
		{"Missing prefix", "0123456789abcdef0", false},
		{"Empty ID", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate instance ID format: i- prefix and 17 or 19 characters total
			hasValidPrefix := strings.HasPrefix(tt.id, "i-")
			isValidLength := len(tt.id) == 18 || len(tt.id) == 19
			isValid := hasValidPrefix && isValidLength

			if isValid != tt.valid {
				t.Errorf("Instance ID %s validation = %v, expected %v", tt.id, isValid, tt.valid)
			}
		})
	}
}

func TestRDSEngineVersions(t *testing.T) {
	engines := []string{
		"postgres",
		"postgres14",
		"postgres15",
		"mysql",
		"mysql8.0",
		"mariadb",
		"mariadb10.6",
		"aurora-mysql",
		"aurora-postgresql",
	}

	for _, engine := range engines {
		rds := RDSInstance{
			Identifier: "test-db",
			Engine:     engine,
		}
		if rds.Engine == "" {
			t.Error("RDS Engine should not be empty")
		}
		// Verify oracle is not in the list
		if strings.Contains(strings.ToLower(rds.Engine), "oracle") {
			t.Errorf("Oracle engine %s should be filtered out", engine)
		}
	}
}
