package utils

import (
	"fmt"
	"log"
	"os"
	"strings"
)

var logFile *os.File

// setupLogFile ensures the logs directory exists and returns a file for logging.
func SetupLogFile() (*os.File, error) {
	// Ensure the logs directory exists
	if _, statErr := os.Stat("logs"); os.IsNotExist(statErr) {
		mkdirErr := os.Mkdir("logs", 0755)
		if mkdirErr != nil {
			return nil, fmt.Errorf("error creating logs directory: %w", mkdirErr)
		}
	}

	file, fileErr := os.OpenFile("logs/application.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if fileErr != nil {
		return nil, fmt.Errorf("error opening log file: %w", fileErr)
	}
	logFile = file

	return logFile, nil
}

// LogInfo logs an informational message to the log file.
func Logger(message string, err error) {

	if strings.Contains(message, "ERROR") {
		logger := log.New(logFile, "", log.Ldate|log.Ltime|log.Lshortfile)
		logger.Println(message)
		terminalLogger := log.New(os.Stderr, "", log.Ldate|log.Ltime|log.Lshortfile)
		terminalLogger.Println(fmt.Sprintf("%s: %v", message, err))
	} else {
		logger := log.New(logFile, "", log.Ldate|log.Ltime|log.Lshortfile)
		logger.Println(message)
	}
}
