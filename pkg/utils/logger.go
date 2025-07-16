package utils

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// Formats and prints a log message in the application's standard format.
func StratumLog(level string, format string, args ...interface{}) {
	fmt.Fprintf(gin.DefaultWriter, "[STRATUM] %s | %-5s | %s\n",
		time.Now().Format("2006/01/02 - 15:04:05"),
		level,
		fmt.Sprintf(format, args...),
	)
}
