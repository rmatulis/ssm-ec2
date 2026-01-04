# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release with EC2 SSM connection support
- RDS IAM authentication token generation
- Interactive mode selection
- AWS profile and region support
- Multi-platform binary builds via GitHub Actions
- Automatic version bumping and releases

### Features
- List and connect to EC2 instances via AWS SSM Session Manager
- List RDS instances and generate IAM auth tokens
- Support for MySQL, MariaDB, and PostgreSQL databases
- Username validation for RDS connections
- Filtering of Oracle databases (no IAM auth support)
