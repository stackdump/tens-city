package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/stackdump/tens-city/pkg/activitypub"
	"github.com/stackdump/tens-city/pkg/docserver"
	"github.com/stackdump/tens-city/pkg/logger"
	"github.com/stackdump/tens-city/pkg/static"
	"github.com/stackdump/tens-city/pkg/webserver"
)

func main() {
	addr := flag.String("addr", ":8080", "Server address")
	storeDir := flag.String("store", "data", "Filesystem store directory")
	contentDir := flag.String("content", "content/posts", "Content directory for markdown blog posts")
	baseURL := flag.String("base-url", "http://localhost:8080", "Base URL for the server")
	indexLimit := flag.Int("index-limit", 20, "Maximum number of posts to show in index (0 = no limit)")
	jsonlLog := flag.Bool("jsonl", false, "Use JSONL format for logging")
	logHeaders := flag.Bool("log-headers", false, "Log incoming request headers (useful for debugging RSS http/https behavior)")
	flag.Parse()

	// Check for INDEX_LIMIT environment variable (overrides flag default)
	if envLimit := os.Getenv("INDEX_LIMIT"); envLimit != "" {
		if limit, err := strconv.Atoi(envLimit); err == nil {
			indexLimit = &limit
		} else {
			log.Printf("Warning: Invalid INDEX_LIMIT environment variable '%s': %v. Using default or flag value.", envLimit, err)
		}
	}

	// Check for GOOGLE_ANALYTICS_ID environment variable
	googleAnalyticsID := os.Getenv("GOOGLE_ANALYTICS_ID")

	// Create logger based on format
	var appLogger logger.Logger
	if *jsonlLog {
		appLogger = logger.NewJSONLLogger(os.Stdout)
		appLogger.LogInfo("Using JSONL logging format")
	} else {
		appLogger = logger.NewTextLogger()
	}

	appLogger.LogInfo(fmt.Sprintf("Using filesystem storage: %s", *storeDir))
	appLogger.LogInfo(fmt.Sprintf("Content directory: %s", *contentDir))
	appLogger.LogInfo(fmt.Sprintf("Fallback Base URL: %s", *baseURL))
	appLogger.LogInfo(fmt.Sprintf("Index limit: %d", *indexLimit))
	appLogger.LogInfo(fmt.Sprintf("Header logging: %v", *logHeaders))
	if googleAnalyticsID != "" {
		appLogger.LogInfo(fmt.Sprintf("Google Analytics ID: %s", googleAnalyticsID))
	}
	storage := webserver.NewFSStorage(*storeDir)

	// Get the embedded public filesystem
	publicSubFS, err := static.Public()
	if err != nil {
		log.Fatalf("Failed to access embedded public files: %v", err)
	}

	// Create document server with fallback URL
	docServer := docserver.NewDocServer(*contentDir, *baseURL, *indexLimit, googleAnalyticsID)

	// Initialize ActivityPub actor if configured via environment variables
	// Required: ACTIVITYPUB_DOMAIN, ACTIVITYPUB_USERNAME
	// Optional: ACTIVITYPUB_DISPLAY_NAME, ACTIVITYPUB_SUMMARY
	var actor *activitypub.Actor
	apDomain := os.Getenv("ACTIVITYPUB_DOMAIN")
	apUsername := os.Getenv("ACTIVITYPUB_USERNAME")

	if apDomain != "" && apUsername != "" {
		// Get optional config
		apDisplayName := os.Getenv("ACTIVITYPUB_DISPLAY_NAME")
		if apDisplayName == "" {
			apDisplayName = apUsername
		}
		apSummary := os.Getenv("ACTIVITYPUB_SUMMARY")
		apProfileURL := os.Getenv("ACTIVITYPUB_PROFILE_URL")
		if apProfileURL == "" {
			apProfileURL = "https://" + apDomain + "/"
		}
		apIconURL := os.Getenv("ACTIVITYPUB_ICON_URL")
		if apIconURL == "" {
			apIconURL = "https://" + apDomain + "/favicon.svg"
		}

		// Key storage path - default to data directory
		keyPath := os.Getenv("ACTIVITYPUB_KEY_PATH")
		if keyPath == "" {
			keyPath = filepath.Join(*storeDir, "activitypub.key")
		}

		config := &activitypub.Config{
			Username:        apUsername,
			Domain:          apDomain,
			DisplayName:     apDisplayName,
			Summary:         apSummary,
			ProfileURL:      apProfileURL,
			IconURL:         apIconURL,
			KeyPath:         keyPath,
			SoftwareName:    "tens-city",
			SoftwareVersion: "1.0.0",
		}

		actor, err = activitypub.NewActor(config)
		if err != nil {
			log.Printf("Warning: Failed to initialize ActivityPub actor: %v", err)
			log.Printf("ActivityPub federation will be disabled")
		} else {
			appLogger.LogInfo(fmt.Sprintf("ActivityPub enabled: @%s@%s", apUsername, apDomain))
		}
	}

	server := webserver.NewServer(storage, publicSubFS, docServer, *baseURL, googleAnalyticsID, actor, *contentDir)

	// Wrap server with logging middleware
	handler := logger.LoggingMiddleware(appLogger, *logHeaders)(server)

	appLogger.LogInfo(fmt.Sprintf("Starting server on %s", *addr))
	appLogger.LogInfo("Using embedded public files")
	appLogger.LogInfo("Server will detect protocol from proxy headers (X-Forwarded-Proto, X-Forwarded-Scheme, X-Forwarded-Ssl, Forwarded)")
	if err := http.ListenAndServe(*addr, handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
