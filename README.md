# ssm-ec2

A command-line tool for connecting to AWS EC2 instances via AWS Systems Manager (SSM) Session Manager with an interactive instance selector.

## Features

- 🔍 **Interactive Instance Selection** - Browse and select from your SSM-managed EC2 instances using an interactive menu
- 🏷️ **Rich Instance Information** - Displays instance ID, Name tag, computer name, and platform type
- 🔒 **Secure Sessions** - Connects via SSM Session Manager (no SSH keys or open ports required)
- 🌏 **Multi-Region Support** - Defaults to `ap-southeast-2` but supports any AWS region
- 👤 **AWS Profile Support** - Use different AWS profiles for multi-account access
- ✨ **User-Friendly** - Built with [Charm](https://github.com/charmbracelet/huh) for a modern CLI experience

## Prerequisites

1. **AWS Session Manager Plugin** - Required for establishing SSM sessions
   ```bash
   # macOS (using Homebrew)
   brew install --cask session-manager-plugin
   
   # Or download from AWS
   # https://docs.aws.amazon.com/systems-manager/latest/userguide/session-manager-working-with-install-plugin.html
   ```

2. **AWS Credentials** - Configure your AWS credentials and profiles
   ```bash
   aws configure
   ```

3. **Go 1.21+** (for building from source)

## Installation

### Build from Source

```bash
git clone https://github.com/rmatulis/ssm-ec2.git
cd ssm-ec2
go build -o ssm-ec2 main.go
```

### Install to PATH (optional)

```bash
# macOS/Linux
sudo mv ssm-ec2 /usr/local/bin/

# Or add to your home bin directory
mv ssm-ec2 ~/bin/
```

## Usage

### Basic Usage

```bash
# Use default region (ap-southeast-2) and default profile
./ssm-ec2

# Specify a different region
./ssm-ec2 -region us-east-1

# Use a specific AWS profile
./ssm-ec2 -profile production

# Combine options
./ssm-ec2 -region eu-west-1 -profile dev-account
```

### Command-line Options

| Flag | Default | Description |
|------|---------|-------------|
| `-region` | `ap-southeast-2` | AWS region to target |
| `-profile` | `default` | AWS profile to use |

### Example Session

```bash
$ ./ssm-ec2
2025/11/05 10:30:15 Using profile: default, region: ap-southeast-2

Select an Instance to Connect
> i-0123456789abcdef0 (web-server-01) - ip-10-0-1-100 [Linux]
  i-0fedcba9876543210 (app-server-01) - ip-10-0-2-50 [Linux]
  i-0abcdef1234567890 (win-server-01) - EC2AMAZ-12345 [Windows]

2025/11/05 10:30:20 Starting SSM session for i-0123456789abcdef0...

Starting session with SessionId: john.doe-0a1b2c3d4e5f6g7h8
sh-4.2$ 
```

## How It Works

1. **Validates Prerequisites** - Checks for `session-manager-plugin` installation
2. **Loads AWS Config** - Uses specified region and profile
3. **Fetches SSM Instances** - Retrieves all online instances managed by SSM
4. **Enriches with EC2 Tags** - Adds EC2 Name tags for better identification
5. **Interactive Selection** - Presents a menu to choose your target instance
6. **Establishes Session** - Starts SSM session and hands off to the plugin

## Requirements for EC2 Instances

For an instance to appear in the list, it must:

- Have the **SSM Agent** installed and running
- Have an **IAM instance profile** with SSM permissions (e.g., `AmazonSSMManagedInstanceCore`)
- Be in an **"Online"** ping status in SSM
- Be in the specified AWS region

## Dependencies

- [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2)
- [Charm Huh](https://github.com/charmbracelet/huh) - Interactive forms and prompts

## Troubleshooting

### No instances found
- Verify instances have SSM agent installed
- Check IAM instance profile has required SSM permissions
- Ensure instances are in the specified region
- Confirm instances show as "Online" in AWS Systems Manager console

### Session Manager Plugin not found
```bash
# Verify installation
which session-manager-plugin

# Install if missing (macOS)
brew install --cask session-manager-plugin
```

### Permission denied
- Check your AWS credentials are configured correctly
- Verify your IAM user/role has SSM permissions (`ssm:StartSession`, `ssm:DescribeInstanceInformation`)
- Ensure you have EC2 read permissions for tag retrieval

## License

MIT

## Contributing

Issues and pull requests are welcome!

