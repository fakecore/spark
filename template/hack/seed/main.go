package main

import (
	"errors"
	"fmt"
	"os"
	"time"

	"spark/internal/common/constants"
	"spark/internal/data/dal/model"
	"spark/pkg/utils"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const defaultDatabaseURL = "postgres://{{.DBUser}}:{{.DBPassword}}@127.0.0.1:5432/{{.DBName}}?sslmode=disable"

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = defaultDatabaseURL
	}

	adminUser := getenvDefault("PROJECT_TEMPLATE_SEED_ADMIN_USER", "admin")
	adminPassword := getenvDefault("PROJECT_TEMPLATE_SEED_ADMIN_PASSWORD", "admin123456")
	adminEmail := getenvDefault("PROJECT_TEMPLATE_SEED_ADMIN_EMAIL", "admin@local")
	adminMobile := getenvDefault("PROJECT_TEMPLATE_SEED_ADMIN_MOBILE", "00000000000")

	roleName := getenvDefault("PROJECT_TEMPLATE_SEED_ADMIN_ROLE", "admin")
	deptName := getenvDefault("PROJECT_TEMPLATE_SEED_ADMIN_DEPT", "Root")

	db, err := gorm.Open(postgres.Open(dbURL), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "connect db failed: %v\n", err)
		os.Exit(1)
	}

	nowMs := time.Now().UnixMilli()
	var createdDept model.SysDept
	var createdRole model.SysRole
	var createdUser model.SysUser

	if err := db.Transaction(func(tx *gorm.DB) error {
		// 1) Dept
		if err := tx.Where("dept_name = ?", deptName).First(&createdDept).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("query dept: %w", err)
			}
			createdDept = model.SysDept{
				ParentID:  0,
				Ancestors: "0",
				DeptName:  deptName,
				Sort:      0,
				Status:    1,
			}
			if err := tx.Create(&createdDept).Error; err != nil {
				return fmt.Errorf("create dept: %w", err)
			}
		}

		// 2) Role
		if err := tx.Where("name = ?", roleName).First(&createdRole).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("query role: %w", err)
			}
			createdRole = model.SysRole{
				Status:    1,
				Sort:      0,
				Name:      roleName,
				Code:      constants.RoleCodeSuperAdmin,
				DataScope: 1,
				IsAdmin:   1,
			}
			if err := tx.Create(&createdRole).Error; err != nil {
				return fmt.Errorf("create role: %w", err)
			}
		}

		// 3) User
		if err := tx.Where("name = ?", adminUser).First(&createdUser).Error; err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("query user: %w", err)
			}

			hashed, err := utils.HashPassword(adminPassword)
			if err != nil {
				return fmt.Errorf("hash password: %w", err)
			}

			createdUser = model.SysUser{
				Name:     adminUser,
				Nickname: adminUser,
				Mobile:   adminMobile,
				Birthday: 0,
				Password: hashed,
				Status:   1,
				Email:    adminEmail,
				Sex:      0,
				Avatar:   "",
				DeptID:   createdDept.ID,
				IsAdmin:  1,
			}
			createdUser.CreatedAt = nowMs
			createdUser.UpdatedAt = nowMs
			if err := tx.Create(&createdUser).Error; err != nil {
				return fmt.Errorf("create user: %w", err)
			}
		}

		// 4) User -> Role mapping via casbin_rule (ptype=g, v0=userID, v1=roleCode)
		ptype := "g"
		v0 := fmt.Sprintf("%d", createdUser.ID)
		v1 := createdRole.Code

		var existing model.CasbinRule
		err := tx.Where("ptype = ? AND v0 = ? AND v1 = ?", ptype, v0, v1).First(&existing).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return fmt.Errorf("query casbin rule: %w", err)
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			rule := model.CasbinRule{
				Ptype: &ptype,
				V0:    &v0,
				V1:    &v1,
			}
			if err := tx.Create(&rule).Error; err != nil {
				return fmt.Errorf("create casbin rule: %w", err)
			}
		}

		return nil
	}); err != nil {
		fmt.Fprintf(os.Stderr, "seed failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("seed ok\n")
	fmt.Printf("  dept: %s (id=%d)\n", deptName, createdDept.ID)
	fmt.Printf("  role: %s (id=%d)\n", roleName, createdRole.ID)
	fmt.Printf("  user: %s (id=%d)\n", adminUser, createdUser.ID)
	if createdUser.ID != 1 {
		fmt.Printf("  note: user id is %d (super-admin bypass is only for id=1)\n", createdUser.ID)
	}
}

func getenvDefault(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
