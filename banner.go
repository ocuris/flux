package flux

import (
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	// Version of Flux
	Version = "1.0.0"
	website = "https://github.com/ocuris/flux"
	// ASCII art generated with http://patorjk.com/software/taag/#p=display&f=ANSI%20Shadow&t=Flux
)

// ANSI color codes - Modern gradient palette
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
	// Modern bright colors
	colorBrightBlue   = "\033[94m"
	colorBrightCyan   = "\033[96m"
	colorBrightGreen  = "\033[92m"
	colorBrightPurple = "\033[95m"
	colorBrightYellow = "\033[93m"
)

// StartupLogger handles beautiful startup logging
type StartupLogger struct {
	config Config
	routes []RouteInfo
}

// RouteInfo holds information about a registered route
type RouteInfo struct {
	Method string
	Path   string
}

// NewStartupLogger creates a new startup logger
func NewStartupLogger(config Config) *StartupLogger {
	return &StartupLogger{
		config: config,
		routes: make([]RouteInfo, 0),
	}
}

// AddRoute adds a route to the logger
func (l *StartupLogger) AddRoute(method, path string) {
	l.routes = append(l.routes, RouteInfo{
		Method: method,
		Path:   path,
	})
}

// PrintStartup prints the beautiful startup message
func (l *StartupLogger) PrintStartup(addr string) {
	// Clear screen effect (optional)
	fmt.Println()

	// Print header
	l.printHeader()

	// Print app info
	l.printAppInfo()

	// Print server info
	l.printServerInfo(addr)

	// Print routes
	// l.printRoutes()

	// Print footer
	l.printFooter(addr)

	fmt.Println()
}

func (l *StartupLogger) printHeader() {
	// Gradient colors for the banner (cyan -> blue -> purple)
	gradientColors := []string{
		"\033[96m", // Bright Cyan
		"\033[94m", // Bright Blue
		"\033[95m", // Bright Purple
	}

	bannerLines := []string{
		"   ███████╗██╗     ██╗   ██╗██╗  ██╗",
		"   ██╔════╝██║     ██║   ██║╚██╗██╔╝",
		"   █████╗  ██║     ██║   ██║ ╚███╔╝ ",
		"   ██╔══╝  ██║     ██║   ██║ ██╔██╗ ",
		"   ██║     ███████╗╚██████╔╝██╔╝ ██╗",
		"   ╚═╝     ╚══════╝ ╚═════╝ ╚═╝  ╚═╝",
	}

	fmt.Println()

	// Print each line with gradient
	for i, line := range bannerLines {
		colorIdx := i * len(gradientColors) / len(bannerLines)
		if colorIdx >= len(gradientColors) {
			colorIdx = len(gradientColors) - 1
		}
		fmt.Printf("%s%s%s%s\n", colorBold, gradientColors[colorIdx], line, colorReset)
	}

	// Version line with gradient
	versionStr := fmt.Sprintf("%s%sv%s%s", colorBold, colorBrightCyan, Version, colorReset)
	fmt.Printf("   %s%s%s\n", gradientColors[2], versionStr, colorReset)

	// Tagline with gradient effect
	fmt.Printf("   %s%sModern, high-performance Go web framework%s\n",
		colorBold, colorBrightBlue, colorReset)

	// Website
	fmt.Printf("   %s%s%s\n", colorBrightCyan, website, colorReset)

	// Separator with gradient
	separator := "   ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
	fmt.Printf("%s%s%s\n", colorBrightCyan, separator, colorReset)

	fmt.Println()
}

func (l *StartupLogger) printAppInfo() {
	fmt.Printf("%s%s▸ Application%s  %s%s%s\n", colorBold, colorBrightPurple, colorReset, colorBold, l.config.Title, colorReset)
	fmt.Printf("%s%s▸ Description%s %s\n", colorBold, colorBrightPurple, colorReset, l.config.Description)
	fmt.Printf("%s%s▸ Version%s      %s%s%s\n", colorBold, colorBrightPurple, colorReset, colorBrightCyan, l.config.Version, colorReset)
	fmt.Println()
}

func (l *StartupLogger) printServerInfo(addr string) {
	// Parse address
	host := "localhost"
	port := "8000"
	if strings.Contains(addr, ":") {
		parts := strings.Split(addr, ":")
		if parts[0] != "" {
			host = parts[0]
		}
		if len(parts) > 1 {
			port = parts[1]
		}
	}

	fmt.Printf("%s%s⚡ Server%s\n", colorBold, colorBrightGreen, colorReset)
	fmt.Printf("   %s╰─▸%s Running at   %s%shttp://%s:%s%s\n", colorDim, colorReset, colorBold, colorBrightBlue, host, port, colorReset)
	fmt.Printf("   %s╰─▸%s Started at   %s%s%s\n", colorDim, colorReset, colorDim, time.Now().Format("2006-01-02 15:04:05"), colorReset)
	fmt.Printf("   %s╰─▸%s Environment  %s%s%s\n", colorDim, colorReset, colorBrightYellow, getEnv(), colorReset)
	fmt.Println()
}

// func (l *StartupLogger) printRoutes() {
// 	if len(l.routes) == 0 {
// 		return
// 	}

// 	fmt.Printf("%s%s◆ Routes%s\n", colorBold, colorBrightPurple, colorReset)

// 	// Group routes by path
// 	routeMap := make(map[string][]string)
// 	for _, route := range l.routes {
// 		routeMap[route.Path] = append(routeMap[route.Path], route.Method)
// 	}

// 	// Print routes
// 	count := 0
// 	for path, methods := range routeMap {
// 		count++
// 		isLast := count == len(routeMap)
// 		connector := "├▸"
// 		if isLast {
// 			connector = "╰▸"
// 		}

// 		methodStr := ""
// 		for i, method := range methods {
// 			if i > 0 {
// 				methodStr += ", "
// 			}
// 			methodStr += l.colorizeMethod(method)
// 		}

// 		fmt.Printf("   %s%s%s %s%-50s%s %s\n",
// 			colorDim, connector, colorReset,
// 			colorDim, path, colorReset,
// 			methodStr)
// 	}
// 	fmt.Println()
// }

func (l *StartupLogger) printFooter(addr string) {
	// Parse address for docs URL
	host := "localhost"
	port := "8000"
	if strings.Contains(addr, ":") {
		parts := strings.Split(addr, ":")
		if parts[0] != "" {
			host = parts[0]
		}
		if len(parts) > 1 {
			port = parts[1]
		}
	}

	docsURL := fmt.Sprintf("http://%s:%s/docs", host, port)
	openapiURL := fmt.Sprintf("http://%s:%s/openapi.json", host, port)

	fmt.Printf("%s%s◆ Documentation%s\n", colorBold, colorBrightCyan, colorReset)
	fmt.Printf("   %s├▸%s Interactive docs  %s%s%s%s\n", colorDim, colorReset, colorBold, colorBrightBlue, docsURL, colorReset)
	fmt.Printf("   %s╰▸%s OpenAPI schema   %s%s%s%s\n", colorDim, colorReset, colorBold, colorBrightBlue, openapiURL, colorReset)
	fmt.Println()

	fmt.Printf("%s%s◆ Quick Start%s\n", colorBold, colorBrightYellow, colorReset)
	fmt.Printf("   %s▸%s Press %sCTRL+C%s to stop the server\n", colorDim, colorReset, colorBold, colorReset)
	fmt.Printf("   %s▸%s Visit %s/docs%s for interactive API documentation\n", colorDim, colorReset, colorBold, colorReset)
	fmt.Printf("   %s▸%s Check %s/openapi.json%s for the OpenAPI specification\n", colorDim, colorReset, colorBold, colorReset)
}

// func (l *StartupLogger) colorizeMethod(method string) string {
// 	switch method {
// 	case "GET":
// 		return fmt.Sprintf("%s%s%-6s%s", colorBold, colorGreen, method, colorReset)
// 	case "POST":
// 		return fmt.Sprintf("%s%s%-6s%s", colorBold, colorBlue, method, colorReset)
// 	case "PUT":
// 		return fmt.Sprintf("%s%s%-6s%s", colorBold, colorYellow, method, colorReset)
// 	case "DELETE":
// 		return fmt.Sprintf("%s%s%-6s%s", colorBold, colorRed, method, colorReset)
// 	case "PATCH":
// 		return fmt.Sprintf("%s%s%-6s%s", colorBold, colorPurple, method, colorReset)
// 	default:
// 		return fmt.Sprintf("%s%-6s%s", colorWhite, method, colorReset)
// 	}
// }

func getEnv() string {
	env := os.Getenv("ENV")
	if env == "" {
		env = "development"
	}
	return env
}

// Request logger for middleware
// func logRequest(method, path string, status int, duration time.Duration) {
// 	statusColor := colorGreen
// 	if status >= 400 && status < 500 {
// 		statusColor = colorYellow
// 	} else if status >= 500 {
// 		statusColor = colorRed
// 	}

// 	methodColor := colorCyan
// 	switch method {
// 	case "GET":
// 		methodColor = colorGreen
// 	case "POST":
// 		methodColor = colorBlue
// 	case "PUT":
// 		methodColor = colorYellow
// 	case "DELETE":
// 		methodColor = colorRed
// 	case "PATCH":
// 		methodColor = colorPurple
// 	}

// 	fmt.Printf("%s%s %s%-6s%s %s%s%-50s%s %s%s%d%s %s%6.2fms%s\n",
// 		colorDim,
// 		time.Now().Format("15:04:05"),
// 		methodColor,
// 		method,
// 		colorReset,
// 		colorDim,
// 		colorWhite,
// 		path,
// 		colorReset,
// 		colorBold,
// 		statusColor,
// 		status,
// 		colorReset,
// 		colorDim,
// 		float64(duration.Microseconds())/1000.0,
// 		colorReset,
// 	)
// }
