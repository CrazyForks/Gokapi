package cliflags

import (
	"fmt"
	"github.com/forceu/gokapi/cmd/cli-uploader/cliconstants"
	"github.com/forceu/gokapi/internal/environment"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	// ModeLogin is the mode for the login command
	ModeLogin = iota
	// ModeLogout is the mode for the logout command
	ModeLogout
	// ModeUpload is the mode for the upload command
	ModeUpload
	// ModeArchive is the mode for the archive command
	ModeArchive
	// ModeInvalid is the mode for an invalid command
	ModeInvalid
)

const version = "v1.0.0"

// UploadConfig contains the parameters for the upload command.
type UploadConfig struct {
	File            string
	Directory       string
	TmpFolder       string
	FileName        string
	JsonOutput      bool
	DisableE2e      bool
	ExpiryDays      int
	ExpiryDownloads int
	Password        string
}

// Parse parses the command line arguments and returns the mode.
func Parse() int {
	if len(os.Args) < 2 {
		printUsage(3)
		return ModeInvalid
	}
	switch os.Args[1] {
	case "login":
		return ModeLogin
	case "logout":
		return ModeLogout
	case "upload":
		return ModeUpload
	case "upload-dir":
		return ModeArchive
	case "help":
		printUsage(0)
	default:
		printUsage(3)
	}
	return ModeInvalid
}

// GetUploadParameters parses the command line arguments and returns the parameters for the upload command.
func GetUploadParameters(isArchive bool) UploadConfig {
	result := UploadConfig{}
	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-j":
			fallthrough
		case "--json":
			result.JsonOutput = true
		case "-x":
			fallthrough
		case "--disable-e2e":
			result.DisableE2e = true
		case "-f":
			fallthrough
		case "--file":
			result.File = getParameter(&i)
		case "-e":
			fallthrough
		case "--expiry-days":
			result.ExpiryDays = requireInt(getParameter(&i))
		case "-d":
			fallthrough
		case "--expiry-downloads":
			result.ExpiryDownloads = requireInt(getParameter(&i))
		case "-p":
			fallthrough
		case "--password":
			result.Password = getParameter(&i)
		case "-D":
			fallthrough
		case "--directory":
			result.Directory = getParameter(&i)
		case "-t":
			fallthrough
		case "--tempfolder":
			result.TmpFolder = getParameter(&i)
		case "-n":
			fallthrough
		case "--name":
			result.FileName = getParameter(&i)
		case "-h":
			fallthrough
		case "--help":
			printUsage(0)
		}
	}
	if result.ExpiryDownloads < 0 {
		result.ExpiryDownloads = 0
	}
	if result.ExpiryDays < 0 {
		result.ExpiryDays = 0
	}
	sanitiseFilename(&result)
	if !checkRequiredUploadParameter(&result, isArchive) {
		os.Exit(2)
	}

	return result
}

func sanitiseFilename(config *UploadConfig) {
	if config.FileName == "" {
		return
	}
	config.FileName = filepath.Base(config.FileName)
	config.FileName = strings.TrimSpace(config.FileName)

	// Replace illegal characters with underscore
	// (Windows forbids <>:"/\|?* and control chars)
	illegalChars := regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)
	config.FileName = illegalChars.ReplaceAllString(config.FileName, "_")
}

func checkRequiredUploadParameter(config *UploadConfig, isArchive bool) bool {
	if isArchive && config.Directory != "" {
		return true
	}
	if !isArchive && config.File != "" {
		return true
	}

	if !environment.IsDockerInstance() {
		if isArchive {
			fmt.Println("ERROR: Missing parameter --directory")
		} else {
			fmt.Println("ERROR: Missing parameter --file")
		}
		return false
	}

	ok, uploadPath := getDockerUpload(isArchive)
	if !ok {
		if isArchive {
			fmt.Println("ERROR: Missing parameter --file and no file found in " + cliconstants.DockerFolderUpload)
		} else {
			fmt.Println("ERROR: Missing parameter --file and no file or more than one file found in " + cliconstants.DockerFolderUpload)
		}
		return false
	}

	if isArchive {
		config.File = cliconstants.DockerFolderUpload
	} else {
		config.File = uploadPath
	}
	return true
}

func getDockerUpload(isArchive bool) (bool, string) {
	if !environment.IsDockerInstance() {
		return false, ""
	}
	entries, err := os.ReadDir(cliconstants.DockerFolderUpload)
	if err != nil {
		return false, ""
	}

	var fileName string
	var fileWasFound bool
	for _, entry := range entries {
		if entry.Type().IsRegular() {
			if isArchive {
				return true, cliconstants.DockerFolderUpload
			}
			if fileWasFound {
				// More than one file exist
				return false, ""
			}
			fileName = entry.Name()
			fileWasFound = true
		}
	}
	if !fileWasFound {
		return false, ""
	}
	return true, filepath.Join(cliconstants.DockerFolderUpload, fileName)
}

// GetConfigLocation returns the path to the configuration file. Returns true if the default file is used
func GetConfigLocation() (string, bool) {
	for i := 2; i < len(os.Args); i++ {
		switch os.Args[i] {
		case "-c":
			fallthrough
		case "--configuration":
			return getParameter(&i), false
		}
	}
	if environment.IsDockerInstance() {
		return cliconstants.DockerFolderConfigFile, true
	}
	return cliconstants.DefaultConfigFileName, true
}

func getParameter(position *int) string {
	newPosition := *position + 1
	position = &newPosition
	if newPosition >= len(os.Args) {
		printUsage(3)
	}
	return os.Args[newPosition]
}

func requireInt(input string) int {
	result, err := strconv.Atoi(input)
	if err != nil {
		fmt.Println("ERROR: " + input + " is not a valid integer")
		os.Exit(2)
	}
	return result
}

func printUsage(exitCode int) {
	fmt.Println("Gokapi CLI " + version)
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  gokapi-cli [command] [options]")
	fmt.Println()

	fmt.Println("Commands:")
	fmt.Println("  login          Save login credentials")
	fmt.Println("  upload         Upload a file to the Gokapi instance")
	fmt.Println("  upload-dir     Upload a folder as a zip file to the Gokapi instance")
	fmt.Println("  logout         Delete login credentials")
	fmt.Println()

	fmt.Println("Options:")
	fmt.Println("  -f, --file <path>               File to upload (required for \"upload\")")
	fmt.Println("  -D, --directory <path>          Folder to upload (required for \"upload-dir\")")
	fmt.Println("  -c, --configuration <path>      Path to configuration file (default: gokapi-cli.json)")
	fmt.Println("  -j, --json                      Output the result in JSON only")
	fmt.Println("  -x, --disable-e2e               Disable end-to-end encryption")
	fmt.Println("  -e, --expiry-days <int>         Set file expiry in days (default: unlimited)")
	fmt.Println("  -d, --expiry-downloads <int>    Set max allowed downloads (default: unlimited)")
	fmt.Println("  -p, --password <string>         Set a password for the file")
	fmt.Println("  -n, --name <string>             Change final filename for uploaded file")
	fmt.Println("  -t, --tmpfolder <path>          Folder for temporary Zip file when uploading a directory")
	fmt.Println("  -h, --help                      Show this help message")
	fmt.Println()

	fmt.Println("Examples:")
	fmt.Println("  gokapi-cli login")
	fmt.Println("  gokapi-cli logout -c /path/to/config")
	fmt.Println("  gokapi-cli upload -f /file/to/upload --expiry-days 7 --json")
	fmt.Println("  gokapi-cli upload-dir -D /path/to/upload -t /mnt/tmp")
	fmt.Println()
	os.Exit(exitCode)
}
