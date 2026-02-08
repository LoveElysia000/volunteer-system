package main

import (
	"flag"
	"log"
	"os"

	"volunteer-system/cmd/cli"
)

func main() {
	// 定义命令行参数
	command := flag.String("c", "server", "Command to execute: server (default), version, help")
	flag.Parse()

	switch *command {
	case "server":
		// 启动服务器
		cli.StartServer()
	case "version":
		log.Println("Volunteer System v1.0.0")
	case "help":
		printHelp()
	default:
		log.Printf("Unknown command: %s", *command)
		printHelp()
		os.Exit(1)
	}
}

func printHelp() {
	log.Println("Usage: volunteer-system [options]")
	log.Println("Options:")
	log.Println("  -c command    Command to execute: server, version, help")
	log.Println("")
	log.Println("Commands:")
	log.Println("  server        Start the HTTP server (default)")
	log.Println("  version       Show version information")
	log.Println("  help          Show this help message")
}
