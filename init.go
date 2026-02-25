package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the template variables configuration
type Config struct {
	Variables []Variable    `yaml:"variables"`
}

// Variable represents a single template variable
type Variable struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Prompt      string `yaml:"prompt"`
	Default     string `yaml:"default"`
	Env         string `yaml:"env"`
}

// Values holds the actual values for variables
type Values struct {
	ProjectName      string
	ModuleName       string
	DBName           string
	DBUser           string
	DBPassword       string
	DBPort           string
	RedisPort        string
	BackendPort      string
	FrontendPort     string
	Description      string
	Author           string
	Year             string
	Version          string
	// Computed values
	ProjectNameKebab string
}

func toKebabCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '-')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
}

// Replacements returns the replacement map
func (v *Values) Replacements() map[string]string {
	return map[string]string{
		// Project info
		"project-template":   v.ModuleName,
		"projecttemplate":    v.ProjectNameKebab,
		"DoorX":              v.ProjectName,
		"My awesome project": v.Description,

		// Database
		"POSTGRES_USER:-projecttemplate":        "POSTGRES_USER:-" + v.DBName,
		"POSTGRES_PASSWORD:-projecttemplate":   "POSTGRES_PASSWORD:-" + v.DBPassword,
		"POSTGRES_DB:-projecttemplate":         "POSTGRES_DB:-" + v.DBName,
		"user=projecttemplate password=projecttemplate dbname=projecttemplate": "user=" + v.DBUser + " password=" + v.DBPassword + " dbname=" + v.DBName,

		// Docker images
		"projecttemplate-backend": v.ProjectNameKebab + "-backend",
		"projecttemplate-frontend": v.ProjectNameKebab + "-frontend",
		"projecttemplate-infra":    v.ProjectNameKebab + "-infra",

		// Project name in .project
		"projecttemplate\n1.0.0": v.ProjectNameKebab + "\n" + v.Version,
	}
}

func main() {
	var (
		configFile     = flag.String("config", ".template-vars.yaml", "配置文件路径")
		templateDir    = flag.String("t", "template", "模板目录")
		outputDir      = flag.String("o", "", "输出目录 (默认: ./<项目名>)")
		nonInteractive = flag.Bool("y", false, "非交互模式，使用默认值或环境变量")
		showVersion    = flag.Bool("v", false, "显示版本")
		showHelp       = flag.Bool("h", false, "显示帮助")
	)
	flag.Parse()

	if *showHelp {
		printHelp()
		return
	}

	if *showVersion {
		fmt.Println("init v1.0.0")
		return
	}

	// Get project name from argument
	args := flag.Args()
	if len(args) == 0 {
		fmt.Println("错误: 请指定项目名")
		fmt.Println("用法: go run init.go <项目名>")
		os.Exit(1)
	}
	projectName := args[0]

	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取当前目录失败: %v\n", err)
		os.Exit(1)
	}

	// Resolve template directory path
	templateRoot := filepath.Join(cwd, *templateDir)
	if _, err := os.Stat(templateRoot); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "模板目录不存在: %s\n", templateRoot)
		os.Exit(1)
	}

	// Load config
	configPath := filepath.Join(cwd, *configFile)
	cfg, err := loadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载配置失败: %v\n", err)
		os.Exit(1)
	}

	// Collect values
	values, err := collectValues(cfg, projectName, *nonInteractive)
	if err != nil {
		fmt.Fprintf(os.Stderr, "收集变量失败: %v\n", err)
		os.Exit(1)
	}

	// Determine output directory
	if *outputDir == "" {
		*outputDir = filepath.Join(cwd, values.ProjectName)
	}

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "创建输出目录失败: %v\n", err)
		os.Exit(1)
	}

	// Load ignore patterns
	ignorePatterns, err := loadIgnorePatterns(filepath.Join(cwd, ".templateignore"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载 .templateignore 失败: %v\n", err)
		// Continue anyway
	}

	// Process template
	fmt.Printf("正在生成项目: %s\n", values.ProjectName)
	fmt.Printf("模板目录: %s\n", templateRoot)
	fmt.Printf("输出目录: %s\n", *outputDir)

	count, err := processTemplate(templateRoot, *outputDir, values, ignorePatterns)
	if err != nil {
		fmt.Fprintf(os.Stderr, "生成失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n✓ 项目生成完成! 共处理 %d 个文件\n", count)
	fmt.Printf("\n下一步:\n")
	fmt.Printf("  cd %s\n", values.ProjectName)
	fmt.Printf("  make init\n")
	fmt.Printf("  make dev-run\n")
}

func printHelp() {
	fmt.Println("init - 项目模板生成工具\n" +
		"\n用法:" +
		"\n  go run init.go [选项] <项目名>\n" +
		"\n选项:" +
		"\n  -config <文件>   配置文件路径 (默认: .template-vars.yaml)" +
		"\n  -t <目录>        模板目录 (默认: template)" +
		"\n  -o <目录>        输出目录 (默认: ./<项目名>)" +
		"\n  -y               非交互模式，使用默认值或环境变量" +
		"\n  -h               显示帮助" +
		"\n  -v               显示版本\n" +
		"\n示例:" +
		"\n  go run init.go myproject" +
		"\n  go run init.go -o /path/to/output myproject" +
		"\n  go run init.go -y myproject  # 非交互模式\n" +
		"\n环境变量:" +
		"\n  可以使用环境变量设置默认值:" +
		"\n  PROJECT_NAME=myproject go run init.go myapp")
}

func loadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func collectValues(cfg *Config, projectName string, nonInteractive bool) (*Values, error) {
	values := &Values{ProjectName: projectName}

	// First, compute smart defaults based on ProjectName
	projectKebab := toKebabCase(projectName)
	smartDefaults := map[string]string{
		"ModuleName": "github.com/user/" + projectKebab,
		"DBName":     projectKebab,
		"DBUser":     projectKebab,
		"DBPassword": projectKebab,
		"Description": projectName,
	}

	// Collect from env, prompt, or default
	for _, v := range cfg.Variables {
		var val string

		// Check env first
		if v.Env != "" {
			val = os.Getenv(v.Env)
		}

		// Prompt if empty and interactive
		if val == "" && !nonInteractive {
			defaultStr := v.Default
			if defaultStr == "" {
				defaultStr = smartDefaults[v.Name]
				if defaultStr == "" {
					defaultStr = "<空>"
				}
			}
			prompt := v.Prompt
			if prompt == "" {
				prompt = v.Name
			}
			fmt.Printf("%s [%s]: ", prompt, defaultStr)
			reader := bufio.NewReader(os.Stdin)
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input != "" {
				val = input
			}
		}

		// Use smart default, then config default if still empty
		if val == "" {
			if smartDefault := smartDefaults[v.Name]; smartDefault != "" {
				val = smartDefault
			} else if v.Default != "" {
				val = v.Default
			}
		}

		// Set value
		switch v.Name {
		case "ModuleName":
			values.ModuleName = val
		case "DBName":
			values.DBName = val
		case "DBUser":
			values.DBUser = val
		case "DBPassword":
			values.DBPassword = val
		case "DBPort":
			values.DBPort = val
		case "RedisPort":
			values.RedisPort = val
		case "BackendPort":
			values.BackendPort = val
		case "FrontendPort":
			values.FrontendPort = val
		case "Description":
			values.Description = val
		case "Author":
			values.Author = val
		case "Year":
			values.Year = val
		case "Version":
			values.Version = val
		}
	}

	// Set hardcoded defaults for empty values
	if values.DBPort == "" {
		values.DBPort = "5432"
	}
	if values.RedisPort == "" {
		values.RedisPort = "6379"
	}
	if values.BackendPort == "" {
		values.BackendPort = "8080"
	}
	if values.FrontendPort == "" {
		values.FrontendPort = "9001"
	}
	if values.Year == "" {
		values.Year = "2025"
	}
	if values.Version == "" {
		values.Version = "1.0.0"
	}

	// Compute derived values
	values.ProjectNameKebab = toKebabCase(values.ProjectName)

	return values, nil
}

func loadIgnorePatterns(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, err
	}

	var patterns []string
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}

	return patterns, scanner.Err()
}

func isIgnored(relPath string, patterns []string) bool {
	relPath = filepath.ToSlash(relPath)

	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}

		// Handle directory patterns
		if strings.HasSuffix(pattern, "/") {
			if strings.HasPrefix(relPath+"/", pattern) {
				return true
			}
		}

		// Handle glob patterns
		matched, _ := filepath.Match(pattern, filepath.Base(relPath))
		if matched {
			return true
		}

		// Handle path prefix
		if strings.HasPrefix(relPath, pattern) {
			return true
		}
	}

	return false
}

func processTemplate(srcDir, dstDir string, values *Values, ignorePatterns []string) (int, error) {
	count := 0
	replacements := values.Replacements()

	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the source directory itself
		if path == srcDir {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// Check if ignored
		if isIgnored(relPath, ignorePatterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Calculate destination path
		dstPath := filepath.Join(dstDir, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Process file
		return processFile(path, dstPath, replacements, &count)
	})

	return count, err
}

func processFile(srcPath, dstPath string, replacements map[string]string, count *int) error {
	// Read source file
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	// Apply replacements
	content := string(data)
	for old, new := range replacements {
		content = strings.ReplaceAll(content, old, new)
	}

	// Ensure destination directory exists
	dstDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	// Write destination file
	if err := os.WriteFile(dstPath, []byte(content), 0644); err != nil {
		return err
	}

	*count++
	return nil
}
