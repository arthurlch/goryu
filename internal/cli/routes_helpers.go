package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type RouteInfo struct {
	Method     string   `json:"method"`
	Path       string   `json:"path"`
	Handler    string   `json:"handler"`
	Middleware []string `json:"middleware,omitempty"`
	Params     []string `json:"params,omitempty"`
	File       string   `json:"file"`
	Line       int      `json:"line"`
}

func discoverRoutes() ([]RouteInfo, error) {
	var routes []RouteInfo

	searchPaths := []string{
		"internal/handlers",
		"internal/routes",
		"cmd/server",
		"main.go",
	}

	for _, searchPath := range searchPaths {
		if _, err := os.Stat(searchPath); os.IsNotExist(err) {
			continue
		}

		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if !strings.HasSuffix(path, ".go") {
				return nil
			}

			fileRoutes, err := parseRoutesFromFile(path)
			if err != nil {
				return nil
			}

			routes = append(routes, fileRoutes...)
			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	return routes, nil
}

func parseRoutesFromFile(filename string) ([]RouteInfo, error) {
	// try AST parsing for more accurate results
	astRoutes, err := parseRoutesFromAST(filename)
	if err == nil && len(astRoutes) > 0 {
		return astRoutes, nil
	}

	regexRoutes, err := parseRoutesFromRegex(filename)
	if err != nil {
		return nil, err
	}

	return regexRoutes, nil
}

func parseRoutesFromAST(filename string) ([]RouteInfo, error) {
	var routes []RouteInfo

	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filename, nil, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	ast.Inspect(node, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.CallExpr:
			if route := parseCallExprForRoute(fset, x, filename); route != nil {
				routes = append(routes, *route)
			}
		}
		return true
	})

	return routes, nil
}

func parseCallExprForRoute(fset *token.FileSet, call *ast.CallExpr, filename string) *RouteInfo {
	if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
		method := sel.Sel.Name
		if isHTTPMethod(method) && len(call.Args) >= 2 {
			path := extractStringLiteral(call.Args[0])
			handler := extractHandlerName(call.Args[1])

			if path != "" && handler != "" {
				pos := fset.Position(call.Pos())
				return &RouteInfo{
					Method:  strings.ToUpper(method),
					Path:    path,
					Handler: handler,
					File:    filename,
					Line:    pos.Line,
				}
			}
		}
	}

	return nil
}

func extractStringLiteral(expr ast.Expr) string {
	if lit, ok := expr.(*ast.BasicLit); ok && lit.Kind == token.STRING {
		value := lit.Value
		if len(value) >= 2 && value[0] == '"' && value[len(value)-1] == '"' {
			return value[1 : len(value)-1]
		}
	}
	return ""
}

func extractHandlerName(expr ast.Expr) string {
	switch x := expr.(type) {
	case *ast.Ident:
		return x.Name
	case *ast.SelectorExpr:
		if pkg, ok := x.X.(*ast.Ident); ok {
			return pkg.Name + "." + x.Sel.Name
		}
		return x.Sel.Name
	}
	return ""
}

func parseRoutesFromRegex(filename string) ([]RouteInfo, error) {
	var routes []RouteInfo

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`app\.(\w+)\s*\(\s*["']([^"']+)["']\s*,\s*(\w+(?:\.\w+)?)`),
		regexp.MustCompile(`\.(\w+)\s*\(\s*["']([^"']+)["']\s*,\s*(\w+(?:\.\w+)?)`),
		regexp.MustCompile(`router\.(\w+)\s*\(\s*["']([^"']+)["']\s*,\s*(\w+(?:\.\w+)?)`),
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		for _, pattern := range patterns {
			matches := pattern.FindStringSubmatch(line)
			if len(matches) == 4 {
				method := strings.ToUpper(matches[1])
				if isHTTPMethod(method) || method == "USE" {
					routes = append(routes, RouteInfo{
						Method:  method,
						Path:    matches[2],
						Handler: matches[3],
						File:    filename,
						Line:    lineNum,
					})
				}
			}
		}
	}

	return routes, scanner.Err()
}

func isHTTPMethod(method string) bool {
	methods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS", "TRACE", "CONNECT"}
	method = strings.ToUpper(method)
	for _, m := range methods { // ignore linter !
		if method == m {
			return true
		}
	}
	return false
}

func testRoute(path, method string, routes []RouteInfo) []RouteInfo {
	var matches []RouteInfo

	for _, route := range routes {
		if route.Method == method || route.Method == "USE" {
			if matchPath(path, route.Path) {
				params := extractParams(path, route.Path)
				routeCopy := route
				routeCopy.Params = params
				matches = append(matches, routeCopy)
			}
		}
	}

	return matches
}

func matchPath(testPath, routePath string) bool {
	pattern := routePath

	pattern = regexp.MustCompile(`:(\w+)`).ReplaceAllString(pattern, `([^/]+)`)

	pattern = strings.ReplaceAll(pattern, "*", "(.*)")

	if pattern == testPath {
		return true
	}

	regex, err := regexp.Compile("^" + pattern + "$")
	if err != nil {
		return false
	}

	return regex.MatchString(testPath)
}

func extractParams(testPath, routePath string) []string {
	var params []string

	paramRegex := regexp.MustCompile(`:(\w+)`)
	paramNames := paramRegex.FindAllStringSubmatch(routePath, -1)

	pattern := routePath
	pattern = regexp.MustCompile(`:(\w+)`).ReplaceAllString(pattern, `([^/]+)`)

	regex, err := regexp.Compile("^" + pattern + "$")
	if err != nil {
		return params
	}

	matches := regex.FindStringSubmatch(testPath)
	if len(matches) > 1 {
		for i, paramName := range paramNames {
			if i+1 < len(matches) {
				params = append(params, fmt.Sprintf("%s=%s", paramName[1], matches[i+1]))
			}
		}
	}

	return params
}

func findSimilarRoutes(path string, routes []RouteInfo) []RouteInfo {
	var similar []RouteInfo

	for _, route := range routes {
		if isSimilarPath(path, route.Path) {
			similar = append(similar, route)
		}
	}

	return similar
}

func isSimilarPath(path1, path2 string) bool {
	parts1 := strings.Split(strings.Trim(path1, "/"), "/")
	parts2 := strings.Split(strings.Trim(path2, "/"), "/")

	if len(parts1) != len(parts2) {
		return false
	}

	matches := 0
	for i := 0; i < len(parts1); i++ {
		if parts1[i] == parts2[i] || strings.HasPrefix(parts2[i], ":") {
			matches++
		}
	}

	return float64(matches)/float64(len(parts1)) >= 0.5
}

func displayRoutesTable(routes []RouteInfo) error {
	if len(routes) == 0 {
		fmt.Println("\n   No routes to display")
		return nil
	}

	fmt.Println()

	methodWidth := 6
	pathWidth := 4
	handlerWidth := 7

	for _, route := range routes {
		if len(route.Method) > methodWidth {
			methodWidth = len(route.Method)
		}
		if len(route.Path) > pathWidth {
			pathWidth = len(route.Path)
		}
		if len(route.Handler) > handlerWidth {
			handlerWidth = len(route.Handler)
		}
	}

	methodWidth += 2
	pathWidth += 2
	handlerWidth += 2

	fmt.Printf("   %-*s %-*s %-*s %s\n",
		methodWidth, "METHOD",
		pathWidth, "PATH",
		handlerWidth, "HANDLER",
		"FILE")

	fmt.Printf("   %s %s %s %s\n",
		strings.Repeat("-", methodWidth),
		strings.Repeat("-", pathWidth),
		strings.Repeat("-", handlerWidth),
		strings.Repeat("-", 20))

	for _, route := range routes {
		fmt.Printf("   %-*s %-*s %-*s %s:%d\n",
			methodWidth, route.Method,
			pathWidth, route.Path,
			handlerWidth, route.Handler,
			route.File, route.Line)
	}

	fmt.Printf("\n   Total: %d routes\n", len(routes))

	return nil
}

func displayRoutesJSON(routes []RouteInfo) error {
	data, err := json.MarshalIndent(routes, "", "  ")
	if err != nil {
		return err
	}

	fmt.Println(string(data))
	return nil
}
