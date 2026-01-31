package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Parse command line flags
	defaultConfigPath := os.Getenv("MINIFLUX_RULES_FILE")
	if defaultConfigPath == "" {
		defaultConfigPath = "rules.yaml"
	}
	configPath := flag.String("config", defaultConfigPath, "Path to the rules configuration file")
	dryRun := flag.Bool("dry-run", false, "Run without making changes")
	flag.Parse()

	// Setup logger
	logger := log.New(os.Stdout, "[miniflux-jobs] ", log.LstdFlags)

	if *dryRun {
		logger.Println("Dry-run mode enabled: no changes will be applied")
	}

	// Load configuration
	logger.Printf("Loading configuration from %s", *configPath)
	config, err := LoadConfig(*configPath)
	if err != nil {
		logger.Fatalf("Failed to load config: %v", err)
	}
	logger.Printf("Loaded %d rules", len(config.Rules))

	// Get API key
	apiKey, err := GetAPIKey()
	if err != nil {
		logger.Fatalf("Failed to get API key: %v", err)
	}
	logger.Println("API key loaded successfully")

	// Create Miniflux client
	client := NewClientWrapper(config.MinifluxURL, apiKey)

	// Create matcher with compiled rules
	matcher, err := NewMatcher(config.Rules)
	if err != nil {
		logger.Fatalf("Failed to compile rules: %v", err)
	}

	// Create processor
	processor := NewProcessor(client, matcher, logger, *dryRun)

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Run processing loop
	if config.Interval == 0 {
		// Run once and exit
		logger.Println("Running in single-run mode")
		runOnce(processor, logger)
	} else {
		// Run in loop mode
		logger.Printf("Running in loop mode with %d second interval", config.Interval)
		runLoop(processor, logger, config.Interval, sigChan)
	}
}

// runOnce executes a single processing run
func runOnce(processor *Processor, logger *log.Logger) {
	stats, err := processor.Process()
	if err != nil {
		logger.Printf("Processing error: %v", err)
	}
	logStats(logger, stats)
}

// runLoop executes processing in a loop with the given interval
func runLoop(processor *Processor, logger *log.Logger, interval int, sigChan chan os.Signal) {
	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	// Run immediately on start
	logger.Println("Starting initial processing run")
	stats, err := processor.Process()
	if err != nil {
		logger.Printf("Processing error: %v", err)
	}
	logStats(logger, stats)

	for {
		select {
		case <-ticker.C:
			logger.Println("Starting scheduled processing run")
			stats, err := processor.Process()
			if err != nil {
				logger.Printf("Processing error: %v", err)
			}
			logStats(logger, stats)

		case sig := <-sigChan:
			logger.Printf("Received signal %v, shutting down", sig)
			return
		}
	}
}

// logStats logs the processing statistics
func logStats(logger *log.Logger, stats *ProcessStats) {
	logger.Printf(
		"Processing complete: %d entries checked, %d matched, %d marked read, %d removed, %d errors",
		stats.TotalEntries,
		stats.MatchedEntries,
		stats.MarkedRead,
		stats.Removed,
		stats.Errors,
	)
}
