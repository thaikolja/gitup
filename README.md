# GitUp

[![Go Version](https://img.shields.io/badge/go-%3E%3D1.25.4-blue.svg)](https://golang.org/dl/) [![License](https://img.shields.io/badge/license-MIT-green.svg)](https://github.com/thaikolja/gitup/blob/main/LICENSE) [![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20linux%20%7C%20windows-lightgrey.svg)](https://github.com/thaikolja/gitup/releases) [![GitHub stars](https://img.shields.io/github/stars/thaikolja/gitup.svg?style=flat)](https://github.com/thaikolja/gitup/stargazers) [![GitHub issues](https://img.shields.io/github/issues/thaikolja/gitup.svg)](https://github.com/thaikolja/gitup/issues)

**GitUp** makes it easy to share files by uploading them directly to your GitHub repository. The tool handles the complexity of GitHub API integration, file organization, and formatting for you. It's perfect for quickly sharing images, documents, or other files and getting shareable links in return.

## Features

- 🚀 **Simple Upload**: Upload any file with a single command
- 🔐 **Secure Authentication**: Stores GitHub tokens securely in macOS Keychain or config file
- 📁 **Automatic Organization**: Files are automatically placed in appropriate folders (img/, docs/, data/, etc.)
- ✨ **Filename Sanitization**: Converts special characters and spaces to web-friendly formats
- 🔄 **Duplicate Prevention**: Automatically handles duplicate filenames with incremental suffixes
- 📝 **Markdown Output**: Generates appropriate markdown links (image syntax for images)
- 🌐 **Cross-platform**: Works on macOS, Linux, and Windows

## Installation

### Pre-built Binaries (Recommended)

GitUp can be downloaded as a single executable file. Check the [releases page](https://github.com/thaikolja/gitup/releases) for pre-built binaries for your platform.

### From Source

```bash
# Clone the repository
git clone https://github.com/thaikolja/gitup.git
cd gitup

# Build the project
go build -o gitup .

# Or install to your Go bin directory
go install .
```

## Usage

### Configuration

First, configure GitUp with your GitHub credentials:

```bash
./gitup -config
```

This will prompt you for your GitHub Personal Access Token and repository (in `owner/repo` format).

### Uploading Files

Upload a file to your configured repository:

```bash
./gitup path/to/your/file.png
```

Additional options:
- `-v`: Enable verbose logging
- `-branch`: Specify a different branch (default: `main`)
- `-config`: Re-run configuration

Example with options:

```bash
# Upload with verbose output
./gitup -v path/to/your/file.png

# Upload to a specific branch
./gitup -branch development path/to/your/file.png
```

## Examples

```bash
# Upload an image
./gitup screenshot.png
# Output: ![screenshot.png](https://raw.githubusercontent.com/owner/repo/main/img/screenshot.png)

# Upload a document
./gitup document.pdf
# Output: [document.pdf](https://raw.githubusercontent.com/owner/repo/main/docs/document.pdf)

# Upload with verbose output
./gitup -v image.jpg
```

## Author

* [**Kolja Nolte**](https://www.kolja-nolte.com) (kolja.nolte@gmail.com)

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

Made with ❤️ by [Kolja Nolte](https://github.com/thaikolja)