package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

// Config represents the template variables configuration
type Config struct {
	Variables []Variable `yaml:"variables"`
	Derived   []Derived  `yaml:"derived"`
}

// Variable represents a single template variable
type Variable struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
	Prompt      string `yaml:"prompt"`
	Default     string `yaml:"default"`
	Env         string `yaml:"env"`
}

// Derived represents a computed value
type Derived struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
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
	// Derived values
	ProjectNameLower string
	ProjectNameSnake string
	ProjectNameKebab string
	EnvPrefix        string
}

// TemplateFuncs provides custom template functions
func TemplateFuncs() template.FuncMap {
	return template.FuncMap{
		"lower": strings.ToLower,
		"upper": strings.ToUpper,
		"title": strings.Title,
		"toSnake": toSnakeCase,
		"toKebab": toKebabCase,
	}
}

func toSnakeCase(s string) string {
	var result []rune
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result = append(result, '_')
		}
		result = append(result, r)
	}
	return strings.ToLower(string(result))
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

func main() {
	var (
		configFile = flag.String("config", ".template-vars.yaml", "配置文件路径")
		outputDir  = flag.String("o", "", "输出目录 (默认: ./<项目名>)")
		nonInteractive = flag.Bool("y", false, "非交互模式，使用默认值或环境变量")
		showVersion = flag.Bool("v", false, "显示版本")
		showHelp   = flag.Bool("h", false, "显示帮助")
	)
	flag.Parse()

	if *showHelp {
		printHelp()
		return
	}

	if *showVersion {
		fmt.Println("templify v1.0.0")
		return
	}

	// Get project name from argument or prompt
	args := flag.Args()
	var projectName string
	if len(args) > 0 {
		projectName = args[0]
	}

	// Load config
	cfg, err := loadConfig(*configFile)
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
		*outputDir = "./" + values.ProjectName
	}

	// Create output directory
	if err := os.MkdirAll(*outputDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "创建输出目录失败: %v\n", err)
		os.Exit(1)
	}

	// Get template root (current directory)
	templateRoot, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "获取当前目录失败: %v\n", err)
		os.Exit(1)
	}

	// Load ignore patterns
	ignorePatterns, err := loadIgnorePatterns(filepath.Join(templateRoot, ".templateignore"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "加载 .templateignore 失败: %v\n", err)
		// Continue anyway
	}

	// Process template
	fmt.Printf("正在生成项目: %s\n", values.ProjectName)
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
	fmt.Println("templify - 项目模板生成工具\n" +
		"\n用法:" +
		"\n  templify [选项] <项目名>\n" +
		"\n选项:" +
		"\n  -config <文件>   配置文件路径 (默认: .template-vars.yaml)" +
		"\n  -o <目录>        输出目录 (默认: ./<项目名>)" +
		"\n  -y               非交互模式，使用默认值或环境变量" +
		"\n  -h               显示帮助" +
		"\n  -v               显示版本\n" +
		"\n示例:" +
		"\n  templify myproject" +
		"\n  templify myproject -o /path/to/output" +
		"\n  templify myproject -y  # 非交互模式\n" +
		"\n环境变量:" +
		"\n  可以使用环境变量设置默认值:" +
		"\n  PROJECT_NAME=myproject templify new")
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
	values := &Values{}

	// Collect from env, prompt, or default
	for _, v := range cfg.Variables {
		var val string

		// Check env first
		if v.Env != "" {
			val = os.Getenv(v.Env)
		}

		// Check command line for project name
		if v.Name == "ProjectName" && projectName != "" {
			val = projectName
		}

		// Prompt if empty and interactive
		if val == "" && !nonInteractive {
			defaultStr := v.Default
			if defaultStr == "" {
				defaultStr = "<空>"
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

		// Use default if still empty
		if val == "" {
			val = v.Default
		}

		// Set value
		switch v.Name {
		case "ProjectName":
			values.ProjectName = val
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

	// Set defaults for empty values
	if values.ModuleName == "" {
		values.ModuleName = "github.com/user/" + toKebabCase(values.ProjectName)
	}
	if values.DBName == "" {
		values.DBName = toKebabCase(values.ProjectName)
	}
	if values.DBUser == "" {
		values.DBUser = toKebabCase(values.ProjectName)
	}
	if values.DBPassword == "" {
		values.DBPassword = toKebabCase(values.ProjectName)
	}
	if values.Year == "" {
		values.Year = "2025"
	}

	// Compute derived values
	values.ProjectNameLower = strings.ToLower(values.ProjectName)
	values.ProjectNameSnake = toSnakeCase(values.ProjectName)
	values.ProjectNameKebab = toKebabCase(values.ProjectName)
	values.EnvPrefix = strings.ToUpper(values.ProjectName) + "_"

	return values, nil
}

func loadIgnorePatterns(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{
				".git/",
				"cmd/templify/",
				".template-vars.yaml",
				".templateignore",
				"bin/",
				"dist/",
				"*.exe",
				".vscode/",
				".idea/",
				"vendor/",
				"*.sum",
				"docs/",
			}, nil
		}
		return nil, err
	}

	var patterns []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}

	return patterns, scanner.Err()
}

func isIgnored(path string, baseDir string, patterns []string) bool {
	relPath, err := filepath.Rel(baseDir, path)
	if err != nil {
		return false
	}
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
		matched, _ := filepath.Match(pattern, filepath.Base(path))
		if matched {
			return true
		}

		// Handle path prefix
		if strings.HasPrefix(relPath, pattern) {
			return true
		}

		// Handle wildcard patterns
		if strings.Contains(pattern, "*") {
			regex := regexp.QuoteMeta(pattern)
			regex = "^" + strings.ReplaceAll(regex, "\\*", ".*") + "$"
			if m, _ := regexp.MatchString(regex, relPath); m {
				return true
			}
		}
	}

	return false
}

func processTemplate(srcDir, dstDir string, values *Values, ignorePatterns []string) (int, error) {
	count := 0

	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip the source directory itself
		if path == srcDir {
			return nil
		}

		// Check if ignored
		if isIgnored(path, srcDir, ignorePatterns) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Calculate destination path
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// Apply template to directory/file names
		dstPath := filepath.Join(dstDir, relPath)
		dstPath = applyTemplateToPath(dstPath, dstDir, values)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		// Process file
		return processFile(path, dstPath, values, &count)
	})

	return count, err
}

func applyTemplateToPath(path, baseDir string, values *Values) string {
	// Extract filename parts
	dir := filepath.Dir(path)
	base := filepath.Base(path)

	// Apply template to filename
	tmpl, err := template.New("filename").Funcs(TemplateFuncs()).Parse(base)
	if err == nil {
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, values); err == nil {
			base = buf.String()
		}
	}

	return filepath.Join(dir, base)
}

func processFile(srcPath, dstPath string, values *Values, count *int) error {
	// Read source file
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	// Check if file contains template markers
	content := string(data)

	// Try to parse as Go template
	tmpl, err := template.New(filepath.Base(srcPath)).Funcs(TemplateFuncs()).Parse(content)
	if err == nil {
		// It's a template, execute it
		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, values); err != nil {
			// If template execution fails, write original content
			buf.Reset()
			buf.Write(data)
		}
		content = buf.String()
	}

	// Write destination file
	if err := os.WriteFile(dstPath, []byte(content), 0644); err != nil {
		return err
	}

	*count++
	return nil
}
