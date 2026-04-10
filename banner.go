package flux

import (
	"fmt"
	"os"
	"strings"
	"time"
)

// Version is the current Flux framework version.
const (
	Version = "1.0.0"
	website = "https://github.com/ocuris/flux"
)

// ANSI terminal colour codes used by the startup banner and request logger.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorPurple = "\033[35m"
	colorCyan   = "\033[36m"
	colorWhite  = "\033[37m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"

	colorBrightBlue   = "\033[94m"
	colorBrightCyan   = "\033[96m"
	colorBrightGreen  = "\033[92m"
	colorBrightPurple = "\033[95m"
	colorBrightYellow = "\033[93m"
)

// StartupLogger prints the startup banner and server URLs when the app boots.
type StartupLogger struct {
	config Config
	routes []RouteInfo
}

// RouteInfo holds metadata for a single registered route (used by OpenAPI).
type RouteInfo struct {
	Method string
	Path   string
	Doc    *DocBuilder
}

// NewStartupLogger creates a StartupLogger with the given Config.
func NewStartupLogger(config Config) *StartupLogger {
	return &StartupLogger{
		config: config,
		routes: make([]RouteInfo, 0),
	}
}

// AddRoute records a registered route for display at startup.
func (l *StartupLogger) AddRoute(method, path string, doc *DocBuilder) {
	l.routes = append(l.routes, RouteInfo{Method: method, Path: path, Doc: doc})
}

// PrintStartup prints the full startup banner to stdout.
func (l *StartupLogger) PrintStartup(addr string) {
	fmt.Println()
	l.printHeader()
	l.printAppInfo()
	l.printServerInfo(addr)
	l.printFooter(addr)
	fmt.Println()
}

func (l *StartupLogger) printHeader() {
	gradient := []string{
		"\033[96m", // Bright Cyan
		"\033[94m", // Bright Blue
		"\033[95m", // Bright Purple
	}

	lines := []string{
		"   ███████╗██╗     ██╗   ██╗██╗  ██╗",
		"   ██╔════╝██║     ██║   ██║╚██╗██╔╝",
		"   █████╗  ██║     ██║   ██║ ╚███╔╝ ",
		"   ██╔══╝  ██║     ██║   ██║ ██╔██╗ ",
		"   ██║     ███████╗╚██████╔╝██╔╝ ██╗",
		"   ╚═╝     ╚══════╝ ╚═════╝ ╚═╝  ╚═╝",
	}

	fmt.Println()
	for i, line := range lines {
		idx := i * len(gradient) / len(lines)
		if idx >= len(gradient) {
			idx = len(gradient) - 1
		}
		fmt.Printf("%s%s%s%s\n", colorBold, gradient[idx], line, colorReset)
	}

	fmt.Printf("   %s%sv%s%s\n", colorBold, colorBrightCyan, Version, colorReset)
	fmt.Printf("   %s%sModern, high-performance Go web framework%s\n", colorBold, colorBrightBlue, colorReset)
	fmt.Printf("   %s%s%s\n", colorBrightCyan, website, colorReset)
	fmt.Printf("%s   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━%s\n", colorBrightCyan, colorReset)
	fmt.Println()
}

func (l *StartupLogger) printAppInfo() {
	fmt.Printf("%s%s▸ Application%s  %s%s%s\n", colorBold, colorBrightPurple, colorReset, colorBold, l.config.Title, colorReset)
	fmt.Printf("%s%s▸ Description%s %s\n", colorBold, colorBrightPurple, colorReset, l.config.Description)
	fmt.Printf("%s%s▸ Version%s      %s%s%s\n", colorBold, colorBrightPurple, colorReset, colorBrightCyan, l.config.Version, colorReset)
	fmt.Println()
}

func (l *StartupLogger) printServerInfo(addr string) {
	host, port := parseAddr(addr)
	fmt.Printf("%s%s⚡ Server%s\n", colorBold, colorBrightGreen, colorReset)
	fmt.Printf("   %s╰─▸%s Running at   %s%shttp://%s:%s%s\n", colorDim, colorReset, colorBold, colorBrightBlue, host, port, colorReset)
	fmt.Printf("   %s╰─▸%s Started at   %s%s%s\n", colorDim, colorReset, colorDim, time.Now().Format("2006-01-02 15:04:05"), colorReset)
	fmt.Printf("   %s╰─▸%s Environment  %s%s%s\n", colorDim, colorReset, colorBrightYellow, currentEnv(), colorReset)
	fmt.Println()
}

func (l *StartupLogger) printFooter(addr string) {
	host, port := parseAddr(addr)
	docsURL := fmt.Sprintf("http://%s:%s/docs", host, port)
	openapiURL := fmt.Sprintf("http://%s:%s/openapi.json", host, port)

	fmt.Printf("%s%s◆ Documentation%s\n", colorBold, colorBrightCyan, colorReset)
	fmt.Printf("   %s├▸%s Interactive docs  %s%s%s%s\n", colorDim, colorReset, colorBold, colorBrightBlue, docsURL, colorReset)
	fmt.Printf("   %s╰▸%s OpenAPI schema   %s%s%s%s\n", colorDim, colorReset, colorBold, colorBrightBlue, openapiURL, colorReset)
	fmt.Println()

	fmt.Printf("%s%s◆ Quick Start%s\n", colorBold, colorBrightYellow, colorReset)
	fmt.Printf("   %s▸%s Press %sCTRL+C%s to stop the server\n", colorDim, colorReset, colorBold, colorReset)
	fmt.Printf("   %s▸%s Visit %s/docs%s for interactive API documentation\n", colorDim, colorReset, colorBold, colorReset)
}

// logRequest is called by the Logger middleware to print a colourised
// one-line request summary to stdout.
func logRequest(method, path string, status int, duration time.Duration) {
	statusColor := colorGreen
	if status >= 400 && status < 500 {
		statusColor = colorYellow
	} else if status >= 500 {
		statusColor = colorRed
	}

	methodColor := colorCyan
	switch method {
	case "GET":
		methodColor = colorGreen
	case "POST":
		methodColor = colorBlue
	case "PUT":
		methodColor = colorYellow
	case "DELETE":
		methodColor = colorRed
	case "PATCH":
		methodColor = colorPurple
	}

	fmt.Printf("%s%s%s %s%-7s%s %s%-30s%s %s%3d%s %s%.2fms%s\n",
		colorDim, time.Now().Format("15:04:05"), colorReset,
		methodColor, method, colorReset,
		colorWhite, path, colorReset,
		statusColor, status, colorReset,
		colorDim, float64(duration.Microseconds())/1000.0, colorReset,
	)
}

// parseAddr splits a "host:port" or ":port" address string into its components.
func parseAddr(addr string) (host, port string) {
	host, port = "localhost", "8000"
	if strings.Contains(addr, ":") {
		parts := strings.SplitN(addr, ":", 2)
		if parts[0] != "" {
			host = parts[0]
		}
		if parts[1] != "" {
			port = parts[1]
		}
	}
	return
}

// currentEnv returns the value of the ENV environment variable,
// defaulting to "development".
func currentEnv() string {
	if env := os.Getenv("ENV"); env != "" {
		return env
	}
	return "development"
}
