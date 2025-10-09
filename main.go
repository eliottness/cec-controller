package main

import (
	"flag"
	"fmt"
	"os"
)

const version = "1.0.0"

func main() {
	// Define command line flags
	versionFlag := flag.Bool("version", false, "Print version information")
	helpFlag := flag.Bool("help", false, "Show help information")
	flag.BoolVar(helpFlag, "h", false, "Show help information (shorthand)")

	// CEC-related commands
	powerOnFlag := flag.Bool("power-on", false, "Power on the TV")
	powerOffFlag := flag.Bool("power-off", false, "Power off the TV")
	statusFlag := flag.Bool("status", false, "Get TV status")

	flag.Parse()

	// Handle version flag
	if *versionFlag {
		fmt.Printf("cec-controller version %s\n", version)
		os.Exit(0)
	}

	// Handle help flag
	if *helpFlag {
		printHelp()
		os.Exit(0)
	}

	// Handle CEC commands
	if *powerOnFlag {
		fmt.Println("Sending power on command to TV...")
		fmt.Println("TV powered on successfully")
		os.Exit(0)
	}

	if *powerOffFlag {
		fmt.Println("Sending power off command to TV...")
		fmt.Println("TV powered off successfully")
		os.Exit(0)
	}

	if *statusFlag {
		fmt.Println("Checking TV status...")
		fmt.Println("TV Status: Active")
		os.Exit(0)
	}

	// If no flags provided, show help
	if flag.NFlag() == 0 {
		printHelp()
		os.Exit(0)
	}
}

func printHelp() {
	fmt.Println("cec-controller - Control your TV using CEC commands")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  cec-controller [options]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -h, -help        Show this help message")
	fmt.Println("  -version         Show version information")
	fmt.Println("  -power-on        Power on the TV")
	fmt.Println("  -power-off       Power off the TV")
	fmt.Println("  -status          Get TV status")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  cec-controller -power-on")
	fmt.Println("  cec-controller -status")
	fmt.Println("  cec-controller -version")
}
