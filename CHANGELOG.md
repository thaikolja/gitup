# Changelog

All notable changes to this project will be documented in this file. The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## v1.0.0

**Released:** November 27th, 2025

### Added

- Initial release of GitUp
- Command-line interface for uploading files to GitHub repositories
- Secure GitHub token storage using macOS Keychain (with fallback to config file)
- Automatic file categorization by extension (.png/.jpg → /img/, .pdf/.md → /docs/, etc.)
- Filename sanitization and ASCII transliteration
- Duplicate filename handling with incremental suffixes (-1, -2, etc.)
- Support for various file types with appropriate markdown formatting
- Configuration command (`gitup -config`) for setting up credentials
- Verbose logging option
- Branch selection option (default: main)
- File size validation (max 25 MB)
- Automatic markdown link generation for uploaded files
- GitHub API integration for file uploads
- Cross-platform compatibility (macOS, Linux, Windows)

### Security

- Secure token storage using macOS Keychain
- Restrictive file permissions for the config directory and file

[1.0.0]: https://github.com/thaikolja/gitup/releases/tag/v1.0.0