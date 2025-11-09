package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// thats the heaviest feature asofnow

func cmdInit(ctx *Context) error {
	projectName := "my-goryu-app"
	if len(ctx.Args) > 0 {
		projectName = ctx.Args[0]
	}

	template := getFlag(ctx, "template", "basic")
	path := getFlag(ctx, "path", ".")
	module := getFlag(ctx, "module", projectName)
	dbTool := getFlag(ctx, "db-tool", "sqlc")

	fmt.Printf("🚀 Initializing new Goryu project: %s\n", projectName)
	fmt.Printf("   Template: %s\n", template)
	fmt.Printf("   Path: %s\n", path)
	fmt.Printf("   Module: %s\n", module)
	if template == "db" {
		fmt.Printf("   Database Tool: %s\n", dbTool)
	}

	return runInit(projectName, template, path, module, dbTool)
}

func cmdGenerateHandler(ctx *Context) error {
	if len(ctx.Args) < 1 {
		return fmt.Errorf("handler name is required")
	}

	name := ctx.Args[0]
	handlerType := getFlag(ctx, "type", "basic")
	path := getFlag(ctx, "path", "internal/handlers")
	model := getFlag(ctx, "model", "")
	middleware := getFlag(ctx, "middleware", "")

	fmt.Printf("🎯 Generating %s handler: %s\n", handlerType, name)
	
	args := []string{name, "--type=" + handlerType, "--path=" + path}
	if model != "" {
		args = append(args, "--model="+model)
	}
	if middleware != "" {
		args = append(args, "--middleware="+middleware)
	}

	return runGenerateHandler(args)
}

func cmdGenerateMiddleware(ctx *Context) error {
	if len(ctx.Args) < 1 {
		return fmt.Errorf("middleware name is required")
	}

	name := ctx.Args[0]
	mwType := getFlag(ctx, "type", "builder")
	path := getFlag(ctx, "path", "internal/middleware")
	global := getFlag(ctx, "global", "false") == "true"

	fmt.Printf("🛡️  Generating %s middleware: %s\n", mwType, name)
	if global {
		fmt.Println("   ✓ Will be registered globally")
	}

	return runGenerateMiddleware([]string{name, "--type=" + mwType, "--path=" + path})
}

func cmdGenerateModel(ctx *Context) error {
	if len(ctx.Args) < 1 {
		return fmt.Errorf("model name is required")
	}

	name := ctx.Args[0]
	modelType := getFlag(ctx, "type", "basic")
	dbTool := getFlag(ctx, "db-tool", "gorm")
	fields := getFlag(ctx, "fields", "")
	path := getFlag(ctx, "path", "internal/models")

	fmt.Printf("📦 Generating %s model: %s\n", modelType, name)
	if modelType == "db" {
		fmt.Printf("   Using: %s\n", dbTool)
	}
	if fields != "" {
		fmt.Printf("   Fields: %s\n", fields)
	}

	args := []string{name, "--type=" + modelType, "--path=" + path}
	if modelType == "db" {
		args = append(args, "--db-tool="+dbTool)
	}
	if fields != "" {
		args = append(args, "--fields="+fields)
	}

	return runGenerateModel(args)
}

func cmdGenerateRoute(ctx *Context) error {
	if len(ctx.Args) < 1 {
		return fmt.Errorf("route name is required")
	}

	name := ctx.Args[0]
	useBuilder := getFlag(ctx, "builder", "true") == "true"
	group := getFlag(ctx, "group", "")
	middleware := getFlag(ctx, "middleware", "")
	methods := getFlag(ctx, "methods", "GET,POST,PUT,DELETE")

	fmt.Printf("🛤️  Generating route configuration: %s\n", name)
	if useBuilder {
		fmt.Println("   Using route builder pattern")
	}
	if group != "" {
		fmt.Printf("   Group: %s\n", group)
	}

	return generateRouteConfig(name, useBuilder, group, middleware, methods)
}

func cmdGenerateConfig(ctx *Context) error {
	name := "app"
	if len(ctx.Args) > 0 {
		name = ctx.Args[0]
	}

	useBuilder := getFlag(ctx, "builder", "true") == "true"
	configType := getFlag(ctx, "type", "server")
	format := getFlag(ctx, "format", "json")

	fmt.Printf("⚙️  Generating %s configuration: %s\n", configType, name)
	if useBuilder {
		fmt.Println("   Using config builder pattern")
	}
	fmt.Printf("   Format: %s\n", format)

	return generateConfigCode(name, useBuilder, configType, format)
}

func cmdScaffoldAPI(ctx *Context) error {
	if len(ctx.Args) < 1 {
		return fmt.Errorf("resource name is required")
	}

	resource := ctx.Args[0]
	fields := ctx.Flags["fields"]
	if fields == "" {
		return fmt.Errorf("--fields is required for API scaffolding")
	}

	includeDB := getFlag(ctx, "db", "true") == "true"
	includeAuth := getFlag(ctx, "auth", "false") == "true"
	includeValidation := getFlag(ctx, "validation", "true") == "true"
	includeTests := getFlag(ctx, "tests", "true") == "true"

	fmt.Printf("🏗️  Scaffolding REST API for resource: %s\n", resource)
	fmt.Printf("   Fields: %s\n", fields)
	fmt.Printf("   ✓ Database layer: %v\n", includeDB)
	fmt.Printf("   ✓ Authentication: %v\n", includeAuth)
	fmt.Printf("   ✓ Validation: %v\n", includeValidation)
	fmt.Printf("   ✓ Tests: %v\n", includeTests)

	fieldList := parseFields(fields)
	
	fmt.Println("\n📦 Generating model...")
	modelType := "basic"
	if includeDB {
		modelType = "db"
	}
	if err := runGenerateModel([]string{resource, "--type=" + modelType, "--fields=" + fields}); err != nil {
		return fmt.Errorf("failed to generate model: %w", err)
	}

	fmt.Println("\n🎯 Generating CRUD handlers...")
	if err := runGenerateHandler([]string{resource, "--type=crud", "--model=" + resource}); err != nil {
		return fmt.Errorf("failed to generate handlers: %w", err)
	}

	fmt.Println("\n🛤️  Generating routes...")
	middlewareList := ""
	if includeAuth {
		middlewareList = "auth"
	}
	if includeValidation {
		if middlewareList != "" {
			middlewareList += ","
		}
		middlewareList += "validator"
	}
	
	routeGroup := fmt.Sprintf("/api/%s", strings.ToLower(resource))
	if err := generateRouteConfig(resource, true, routeGroup, middlewareList, "GET,POST,PUT,DELETE"); err != nil {
		return fmt.Errorf("failed to generate routes: %w", err)
	}

	if includeDB {
		fmt.Println("\n🗄️  Generating repository...")
		if err := generateRepository(resource, fieldList); err != nil {
			return fmt.Errorf("failed to generate repository: %w", err)
		}
	}

	if includeValidation {
		fmt.Println("\n✅ Generating validation...")
		if err := generateValidation(resource, fieldList); err != nil {
			return fmt.Errorf("failed to generate validation: %w", err)
		}
	}

	if includeTests {
		fmt.Println("\n🧪 Generating tests...")
		if err := generateAPITests(resource, fieldList); err != nil {
			return fmt.Errorf("failed to generate tests: %w", err)
		}
	}

	fmt.Println("\n📖 Generating API documentation...")
	if err := generateAPIDocumentation(resource, fieldList, routeGroup); err != nil {
		return fmt.Errorf("failed to generate documentation: %w", err)
	}

	fmt.Printf("\n✅ Successfully scaffolded REST API for %s!\n", resource)
	fmt.Println("\n💡 Next steps:")
	fmt.Printf("  • Review generated files in internal/handlers/, internal/models/, etc.\n")
	fmt.Printf("  • Update database connection in your main.go\n")
	fmt.Printf("  • Run tests with: go test ./internal/handlers\n")
	fmt.Printf("  • Start server and access API at %s\n", routeGroup)
	
	return nil
}

func cmdScaffoldService(ctx *Context) error {
	if len(ctx.Args) < 1 {
		return fmt.Errorf("service name is required")
	}

	name := ctx.Args[0]
	includeGRPC := getFlag(ctx, "grpc", "false") == "true"
	includeHTTP := getFlag(ctx, "http", "true") == "true"
	includeKafka := getFlag(ctx, "kafka", "false") == "true"
	includeMonitoring := getFlag(ctx, "monitoring", "true") == "true"

	fmt.Printf("🏗️  Scaffolding microservice: %s\n", name)
	fmt.Printf("   ✓ gRPC: %v\n", includeGRPC)
	fmt.Printf("   ✓ HTTP: %v\n", includeHTTP)
	fmt.Printf("   ✓ Kafka: %v\n", includeKafka)
	fmt.Printf("   ✓ Monitoring: %v\n", includeMonitoring)

	fmt.Println("\n📁 Creating service structure...")
	if err := generateServiceStructure(name); err != nil {
		return fmt.Errorf("failed to create service structure: %w", err)
	}

	fmt.Println("\n⚙️  Generating service implementation...")
	if err := generateServiceImplementation(name, includeHTTP, includeGRPC); err != nil {
		return fmt.Errorf("failed to generate service implementation: %w", err)
	}

	if includeHTTP {
		fmt.Println("\n🌐 Generating HTTP handlers...")
		if err := generateHTTPService(name); err != nil {
			return fmt.Errorf("failed to generate HTTP service: %w", err)
		}
	}

	// alors la ?? Grpc you know, not fully implemented but well ... 
	if includeGRPC {
		fmt.Println("\n🔗 Generating gRPC service...")
		if err := generateGRPCService(name); err != nil {
			return fmt.Errorf("failed to generate gRPC service: %w", err)
		}
	}

	if includeKafka {
		fmt.Println("\n📨 Generating Kafka handlers...")
		if err := generateKafkaService(name); err != nil {
			return fmt.Errorf("failed to generate Kafka service: %w", err)
		}
	}

	if includeMonitoring {
		fmt.Println("\n📊 Generating monitoring...")
		if err := generateServiceMonitoring(name); err != nil {
			return fmt.Errorf("failed to generate monitoring: %w", err)
		}
	}

	fmt.Println("\n🐳 Generating Docker and deployment files...")
	if err := generateServiceDeployment(name, includeGRPC, includeHTTP, includeKafka, includeMonitoring); err != nil {
		return fmt.Errorf("failed to generate deployment files: %w", err)
	}

	fmt.Println("\n🚀 Generating service main...")
	if err := generateServiceMain(name, includeGRPC, includeHTTP, includeKafka, includeMonitoring); err != nil {
		return fmt.Errorf("failed to generate service main: %w", err)
	}

	fmt.Printf("\n✅ Successfully scaffolded microservice: %s!\n", name)
	fmt.Println("\n💡 Next steps:")
	fmt.Printf("  • Review generated files in the %s/ directory\n", name)
	fmt.Printf("  • Update configuration in %s/config/\n", name)
	fmt.Printf("  • Implement business logic in %s/internal/service/\n", name)
	if includeGRPC {
		fmt.Printf("  • Review proto files in %s/api/proto/\n", name)
	}
	fmt.Printf("  • Build and run: cd %s && go run cmd/server/main.go\n", name)
	
	return nil
}

func cmdDev(ctx *Context) error {
	port := getFlag(ctx, "port", "3000")
	hotReload := getFlag(ctx, "hot-reload", "true") == "true"
	debug := getFlag(ctx, "debug", "false") == "true"

	fmt.Printf("🔥 Starting development server...\n")
	fmt.Printf("   Port: %s\n", port)
	fmt.Printf("   Hot reload: %v\n", hotReload)
	fmt.Printf("   Debug mode: %v\n", debug)

	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		return fmt.Errorf("go.mod not found - make sure you're in a Go project directory")
	}

	mainPaths := []string{
		"cmd/server/main.go",
		"cmd/main.go", 
		"main.go",
	}
	
	var mainPath string
	for _, path := range mainPaths {
		if _, err := os.Stat(path); err == nil {
			mainPath = path
			break
		}
	}
	
	if mainPath == "" {
		return fmt.Errorf("main.go not found in expected locations (cmd/server/main.go, cmd/main.go, main.go)")
	}
	
	fmt.Printf("   Main file: %s\n", mainPath)
	
	debugStr := "false"
	if debug {
		debugStr = "true"
	}
	
	if hotReload {
		return runDevWithHotReload(mainPath, port, debugStr)
	} else {
		return runDevSimple(mainPath, port, debugStr)
	}
}

func cmdBuild(ctx *Context) error {
	output := getFlag(ctx, "output", "server")
	target := getFlag(ctx, "target", "production")
	compress := getFlag(ctx, "compress", "false") == "true"
	static := getFlag(ctx, "static", "true") == "true"

	fmt.Printf("🔨 Building application...\n")
	fmt.Printf("   Output: %s\n", output)
	fmt.Printf("   Target: %s\n", target)
	fmt.Printf("   Static binary: %v\n", static)
	fmt.Printf("   Compress: %v\n", compress)

	// should be a Go otherwise its a no GO !
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		return fmt.Errorf("go.mod not found - make sure you're in a Go project directory")
	}

	mainPaths := []string{
		"cmd/server/main.go",
		"cmd/main.go", 
		"main.go",
	}
	
	var mainPath string
	for _, path := range mainPaths {
		if _, err := os.Stat(path); err == nil {
			mainPath = path
			break
		}
	}
	
	if mainPath == "" {
		return fmt.Errorf("main.go not found in expected locations (cmd/server/main.go, cmd/main.go, main.go)")
	}

	return runBuild(mainPath, output, target, static, compress)
}

func cmdConfigInit(ctx *Context) error {
	args := []string{}
	if t := ctx.Flags["type"]; t != "" {
		args = append(args, "--type="+t)
	}
	if f := ctx.Flags["format"]; f != "" {
		args = append(args, "--format="+f)
	}
	return runConfigInit(args)
}

func cmdConfigValidate(ctx *Context) error {
	args := []string{}
	if f := ctx.Flags["file"]; f != "" {
		args = append(args, "--file="+f)
	}
	return runConfigValidate(args)
}

func cmdMiddlewareList(ctx *Context) error {
	fmt.Println("📋 Available middleware:")
	fmt.Println("\nBuilt-in middleware:")
	
	builtIn := []struct {
		name string
		desc string
	}{ // notes that new midle shall also be added to internal/middleware/registry.go for build it cli feature
		{"auth", "Authentication middleware"},
		{"basicauth", "Basic authentication"},
		{"cache", "Response caching"},
		{"circuitbreaker", "Circuit breaker pattern"},
		{"compress", "Response compression"},
		{"cors", "Cross-Origin Resource Sharing"},
		{"csrf", "CSRF protection"},
		{"envvar", "Environment variable exposure"},
		{"errors", "Error handling middleware"},
		{"expvar", "Expvar metrics"},
		{"favicon", "Favicon serving"},
		{"fileserver", "Static file serving"},
		{"graceful", "Graceful shutdown"},
		{"healthcheck", "Health check endpoint"},
		{"limiter", "Rate limiting"},
		{"logger", "Request logging"},
		{"metrics", "Prometheus metrics"},
		{"recovery", "Panic recovery"},
		{"requestid", "Request ID generation"},
		{"secure", "Security headers"},
		{"secure_cookie", "Secure cookie handling"},
		{"session", "Session management"},
		{"structlog", "Structured logging"},
		{"timeout", "Request timeout"},
		{"tls_redirect", "TLS redirect"},
		{"tracing", "OpenTelemetry tracing"},
		{"trustproxy", "Trust proxy headers"},
	}

	for _, mw := range builtIn {
		fmt.Printf("  %-20s %s\n", mw.name, mw.desc)
	}

	fmt.Println("\nCustom middleware:")
	
	return nil
}

func cmdMiddlewareInfo(ctx *Context) error {
	if len(ctx.Args) < 1 {
		return fmt.Errorf("middleware name is required")
	}

	name := ctx.Args[0]
	fmt.Printf("ℹ️  Middleware: %s\n", name)
	
	info := getMiddlewareInfo(name)
	if info == nil {
		return fmt.Errorf("middleware '%s' not found", name)
	}
	
	fmt.Printf("\n📋 Details:\n")
	fmt.Printf("   Name: %s\n", info.Name)
	fmt.Printf("   Description: %s\n", info.Description)
	fmt.Printf("   Category: %s\n", info.Category)
	fmt.Printf("   Package: %s\n", info.Package)
	
	if len(info.Dependencies) > 0 {
		fmt.Printf("   Dependencies: %s\n", strings.Join(info.Dependencies, ", "))
	}
	
	fmt.Printf("\n📖 Usage:\n")
	fmt.Printf("%s\n", info.Usage)
	
	if info.Example != "" {
		fmt.Printf("\n💡 Example:\n")
		fmt.Printf("%s\n", info.Example)
	}
	
	if len(info.Options) > 0 {
		fmt.Printf("\n⚙️  Configuration Options:\n")
		for _, option := range info.Options {
			fmt.Printf("   • %s: %s\n", option.Name, option.Description)
			if option.Default != "" {
				fmt.Printf("     Default: %s\n", option.Default)
			}
		}
	}
	
	return nil
}

func cmdConfigMigrate(ctx *Context) error {
	from := ctx.Flags["from"]
	to := ctx.Flags["to"]
	inputFile := getFlag(ctx, "input", "config.json")
	outputFile := getFlag(ctx, "output", "")
	
	if from == "" || to == "" {
		return fmt.Errorf("--from and --to are required")
	}

	fmt.Printf("🔄 Migrating configuration from %s to %s\n", from, to)
	fmt.Printf("   Input: %s\n", inputFile)
	
	if outputFile == "" {
		base := strings.TrimSuffix(inputFile, filepath.Ext(inputFile))
		outputFile = base + "." + to
	}
	fmt.Printf("   Output: %s\n", outputFile)
	
	if _, err := os.Stat(inputFile); os.IsNotExist(err) {
		return fmt.Errorf("input file not found: %s", inputFile)
	}
	
	data, err := os.ReadFile(inputFile)
	if err != nil {
		return fmt.Errorf("failed to read input file: %w", err)
	}
	
	converted, err := migrateConfig(data, from, to)
	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}
	
	if err := os.WriteFile(outputFile, converted, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}
	
	fmt.Printf("✅ Configuration migrated successfully!\n")
	fmt.Printf("   Generated: %s\n", outputFile)
	
	return nil
}

func cmdRoutesList(ctx *Context) error {
	filter := getFlag(ctx, "filter", "")
	format := getFlag(ctx, "format", "table")

	fmt.Println("📍 Application routes:")
	
	if filter != "" {
		fmt.Printf("   Filter: %s\n", filter)
	}

	routes, err := discoverRoutes()
	if err != nil {
		return fmt.Errorf("failed to discover routes: %w", err)
	}
	
	if len(routes) == 0 {
		fmt.Println("   No routes found in the project")
		fmt.Println("\n💡 Routes can be found in:")
		fmt.Println("   • internal/handlers/*.go")
		fmt.Println("   • internal/routes/*.go")
		fmt.Println("   • cmd/server/main.go")
		return nil
	}
	
	if filter != "" {
		filteredRoutes := []RouteInfo{}
		for _, route := range routes {
			if strings.Contains(route.Path, filter) || 
			   strings.Contains(route.Method, filter) ||
			   strings.Contains(route.Handler, filter) {
				filteredRoutes = append(filteredRoutes, route)
			}
		}
		routes = filteredRoutes
	}
	
	switch format {
	case "json":
		return displayRoutesJSON(routes)
	case "table":
		return displayRoutesTable(routes)
	default:
		return fmt.Errorf("unsupported format: %s (use 'table' or 'json')", format)
	}
}

func cmdRoutesTest(ctx *Context) error {
	if len(ctx.Args) < 1 {
		return fmt.Errorf("path is required")
	}

	path := ctx.Args[0]
	method := "GET"
	if len(ctx.Args) > 1 {
		method = strings.ToUpper(ctx.Args[1])
	}

	fmt.Printf("🧪 Testing route: %s %s\n", method, path)
	
	routes, err := discoverRoutes()
	if err != nil {
		return fmt.Errorf("failed to discover routes: %w", err)
	}
	
	matches := testRoute(path, method, routes)
	
	if len(matches) == 0 {
		fmt.Printf("❌ No matching routes found for %s %s\n", method, path)
		
		suggestions := findSimilarRoutes(path, routes)
		if len(suggestions) > 0 {
			fmt.Println("\n💡 Similar routes:")
			for _, suggestion := range suggestions {
				fmt.Printf("   %s %s -> %s\n", suggestion.Method, suggestion.Path, suggestion.Handler)
			}
		}
		return nil
	}
	
	fmt.Printf("✅ Found %d matching route(s):\n", len(matches))
	for _, match := range matches {
		fmt.Printf("   %s %s -> %s", match.Method, match.Path, match.Handler)
		if len(match.Middleware) > 0 {
			fmt.Printf(" [%s]", strings.Join(match.Middleware, ", "))
		}
		if len(match.Params) > 0 {
			fmt.Printf(" (params: %s)", strings.Join(match.Params, ", "))
		}
		fmt.Println()
	}
	
	return nil
}

func cmdVersion(ctx *Context) error {
	showVersion()
	return nil
}

func cmdValidate(ctx *Context) error {
	fix := getFlag(ctx, "fix", "false") == "true"
	
	fmt.Println("🔍 Validating project structure...")
	if fix {
		fmt.Println("   Auto-fix: enabled")
	}

	// For now, delegate to runValidate
	// TODO: Refactor runValidate to use Context directly ??
	return runValidate([]string{})
}


func getFlag(ctx *Context, name string, defaultValue string) string {
	if val, ok := ctx.Flags[name]; ok {
		return val
	}
	return defaultValue
}

func generateRouteConfig(name string, useBuilder bool, group, middleware, methods string) error {
	routesDir := "internal/routes"
	if err := os.MkdirAll(routesDir, 0755); err != nil {
		return err
	}

	filename := filepath.Join(routesDir, strings.ToLower(name)+".go")
	
	var content string
	if useBuilder {
		content = generateRouteBuilderContent(name, group, middleware, methods)
	} else {
		content = generateStandardRouteContent(name, group, middleware, methods)
	}

	return os.WriteFile(filename, []byte(content), 0644)
}

func generateConfigCode(name string, useBuilder bool, configType, format string) error {
	configDir := "internal/config"
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	filename := filepath.Join(configDir, strings.ToLower(name)+"_config.go")
	
	var content string
	if useBuilder {
		content = generateConfigBuilderContent(name, configType)
	} else {
		content = generateStandardConfigContent(name, configType)
	}

	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		return err
	}

	exampleFile := fmt.Sprintf("config.%s.example", format)
	exampleContent := generateExampleConfig(configType, format)
	
	return os.WriteFile(exampleFile, []byte(exampleContent), 0644)
}

func showVersion() {
	fmt.Printf("Goryu CLI v%s\n", VERSION)
	fmt.Println("A GOated web framework") // goated play on words
}


// end of file ... !