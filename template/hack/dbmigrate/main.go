package main

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const defaultDatabaseURL = "postgres://projecttemplate:projecttemplate@127.0.0.1:5432/projecttemplate?sslmode=disable"

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run hack/dbmigrate/main.go [dev|prod|diff|status|reset]")
		fmt.Println("")
		fmt.Println("Commands:")
		fmt.Println("  dev    - 开发环境：智能检测，自动同步")
		fmt.Println("  prod   - 生产环境：使用 Atlas 安全迁移")
		fmt.Println("  diff   - 生成迁移文件 (NAME=xxx)")
		fmt.Println("  status - 查看迁移状态")
		fmt.Println("  reset  - 清空数据库并重新初始化")
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "dev":
		runDevMigrate()
	case "prod":
		runProdMigrate()
	case "diff":
		runDiff()
	case "status":
		runStatus()
	case "reset":
		runReset()
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		os.Exit(1)
	}
}

// runDevMigrate 开发环境：智能检测，自动同步
func runDevMigrate() {
	fmt.Println("=== 开发环境迁移 ===")

	dbURL, err := getDatabaseURL(false)
	if err != nil {
		fmt.Printf("获取 DATABASE_URL 失败: %v\n", err)
		os.Exit(1)
	}

	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		fmt.Printf("连接数据库失败: %v\n", err)
		os.Exit(1)
	}

	// 检查数据库是否为空
	var count int64
	db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name != 'atlas_schema_revisions'").Scan(&count)

	if count == 0 {
		fmt.Println("数据库为空，使用 Atlas 初始化...")
		mustRunAtlas(dbURL, "migrate", "apply", "--env", "local")
	} else {
		fmt.Printf("数据库已有 %d 个表\n", count)

		// 检查是否有待应用的迁移
		if hasPendingMigrations(dbURL) {
			fmt.Println("有待应用的 Atlas 迁移，尝试应用...")
			mustRunAtlas(dbURL, "migrate", "apply", "--env", "local")
		} else {
			fmt.Println("Atlas 无待应用迁移")
			fmt.Println("如需修改字段，请直接修改 model 后执行: make migrate-diff NAME=xxx")
		}
	}
}

// runProdMigrate 生产环境：使用 Atlas，安全可控
func runProdMigrate() {
	fmt.Println("=== 生产环境迁移 ===")
	fmt.Println("使用 Atlas 应用迁移...")
	dbURL, err := getDatabaseURL(true)
	if err != nil {
		fmt.Printf("获取 DATABASE_URL 失败: %v\n", err)
		os.Exit(1)
	}
	mustRunAtlas(dbURL, "migrate", "apply", "--env", "prod")
}

// runDiff 生成迁移文件
func runDiff() {
	name := "auto"
	if len(os.Args) > 2 {
		name = os.Args[2]
	}
	fmt.Printf("=== 生成迁移: %s ===\n", name)
	dbURL, err := getDatabaseURL(false)
	if err != nil {
		fmt.Printf("获取 DATABASE_URL 失败: %v\n", err)
		os.Exit(1)
	}
	mustRunAtlas(dbURL, "migrate", "diff", name, "--env", "local")
}

// runStatus 查看迁移状态
func runStatus() {
	fmt.Println("=== 迁移状态 ===")
	dbURL, err := getDatabaseURL(false)
	if err != nil {
		fmt.Printf("获取 DATABASE_URL 失败: %v\n", err)
		os.Exit(1)
	}
	mustRunAtlas(dbURL, "migrate", "status", "--env", "local")
}

// runReset 重置数据库（清空所有数据）
func runReset() {
	fmt.Println("=== 警告：这将清空数据库所有数据！===")
	fmt.Print("输入 'RESET' 确认: ")
	var input string
	fmt.Scanln(&input)
	if input != "RESET" {
		fmt.Println("取消操作")
		return
	}

	dbURL, err := getDatabaseURL(false)
	if err != nil {
		fmt.Printf("获取 DATABASE_URL 失败: %v\n", err)
		os.Exit(1)
	}

	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{})
	if err != nil {
		fmt.Printf("连接数据库失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("清空数据库...")
	db.Exec("DROP SCHEMA public CASCADE")
	db.Exec("CREATE SCHEMA public")

	fmt.Println("重新生成初始迁移...")
	mustRunAtlas(dbURL, "migrate", "diff", "init", "--env", "local")
	mustRunAtlas(dbURL, "migrate", "apply", "--env", "local")

	fmt.Println("完成！数据库已重置并应用初始迁移。")
}

// hasPendingMigrations 检查是否有待应用的迁移
func hasPendingMigrations(dbURL string) bool {
	atlas, err := findAtlas()
	if err != nil {
		// Defer to mustRunAtlas for a consistent install/help message.
		return true
	}
	cmd := exec.Command(atlas, "migrate", "status", "--env", "local")
	cmd.Dir = "migrations"
	cmd.Env = append(os.Environ(), "DATABASE_URL="+dbURL)
	output, _ := cmd.CombinedOutput()
	return strings.Contains(string(output), "PENDING") ||
		strings.Contains(string(output), "Pending") ||
		strings.Contains(string(output), "待执行")
}

func mustRunAtlas(dbURL string, args ...string) {
	atlas, err := findAtlas()
	if err != nil {
		fmt.Printf("Atlas CLI not found: %v\n", err)
		fmt.Println("Install it with: make ensure-atlas")
		os.Exit(1)
	}
	cmd := exec.Command(atlas, args...)
	cmd.Dir = "migrations"
	cmd.Env = append(os.Environ(), "DATABASE_URL="+dbURL)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Atlas 命令失败: %v\n", err)
		os.Exit(1)
	}
}

func findAtlas() (string, error) {
	// Allow explicit override for CI or non-standard installs.
	if v := os.Getenv("ATLAS_BIN"); v != "" {
		// Ensure it's absolute because we run atlas from a different working directory.
		if filepath.IsAbs(v) {
			return v, nil
		}
		if abs, err := filepath.Abs(v); err == nil {
			return abs, nil
		}
		return v, nil
	}

	// Prefer repo-local pinned atlas installed by `make ensure-atlas`.
	if p := filepath.Join("bin", "atlas"); fileExists(p) {
		return filepath.Abs(p)
	}
	if p := filepath.Join("bin", "atlas.exe"); fileExists(p) {
		return filepath.Abs(p)
	}

	if p, err := exec.LookPath("atlas"); err == nil {
		return p, nil
	}

	// Common Go install locations.
	if gobin := os.Getenv("GOBIN"); gobin != "" {
		if p := filepath.Join(gobin, "atlas"); fileExists(p) {
			return p, nil
		}
	}

	// GOPATH can contain multiple entries.
	out, err := exec.Command("go", "env", "GOPATH").Output()
	if err == nil {
		for _, gp := range strings.Split(strings.TrimSpace(string(out)), string(os.PathListSeparator)) {
			if gp == "" {
				continue
			}
			if p := filepath.Join(gp, "bin", "atlas"); fileExists(p) {
				return p, nil
			}
		}
	}

	return "", fmt.Errorf("atlas not in PATH and not found in ./bin, GOBIN, or GOPATH/bin")
}

func fileExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && !st.IsDir()
}

func getDatabaseURL(strict bool) (string, error) {
	if v := os.Getenv("DATABASE_URL"); v != "" {
		return v, nil
	}
	if strict {
		return "", errors.New("DATABASE_URL is required for prod migration")
	}

	driver, source, err := readConfigDatabase()
	if err == nil && strings.HasPrefix(strings.ToLower(driver), "post") {
		if u, ok := normalizePostgresURL(source); ok {
			return u, nil
		}
	}

	// Safe default for local infra compose.
	return defaultDatabaseURL, nil
}

func readConfigDatabase() (driver, source string, err error) {
	cfgPath := os.Getenv("PROJECT_TEMPLATE_CONFIG_PATH")
	if cfgPath == "" {
		cfgPath = filepath.Join("docker", "backend", "config", "config.yaml")
	}
	b, err := os.ReadFile(cfgPath)
	if err != nil {
		return "", "", err
	}

	var cfg struct {
		Data struct {
			Database struct {
				Driver string `yaml:"driver"`
				Source string `yaml:"source"`
			} `yaml:"database"`
		} `yaml:"data"`
	}
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return "", "", err
	}
	return cfg.Data.Database.Driver, cfg.Data.Database.Source, nil
}

func normalizePostgresURL(source string) (string, bool) {
	s := strings.TrimSpace(source)
	if s == "" {
		return "", false
	}
	ls := strings.ToLower(s)
	if strings.HasPrefix(ls, "postgres://") || strings.HasPrefix(ls, "postgresql://") {
		return s, true
	}

	// Convert "key=value key=value" DSN to postgres:// URL (best-effort).
	kv := map[string]string{}
	for _, part := range strings.Fields(s) {
		k, v, ok := strings.Cut(part, "=")
		if !ok {
			continue
		}
		kv[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
	}
	host := kv["host"]
	user := kv["user"]
	password := kv["password"]
	dbname := kv["dbname"]
	port := kv["port"]
	sslmode := kv["sslmode"]
	if host == "" || user == "" || dbname == "" {
		return "", false
	}
	if port == "" {
		port = "5432"
	}
	if sslmode == "" {
		sslmode = "disable"
	}
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   net.JoinHostPort(host, port),
		Path:   "/" + dbname,
	}
	q := u.Query()
	q.Set("sslmode", sslmode)
	u.RawQuery = q.Encode()
	return u.String(), true
}
