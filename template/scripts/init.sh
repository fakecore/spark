#!/bin/bash

# 项目初始化脚本
# 用法: ./scripts/init.sh <新项目名>

set -e

# 检查参数
if [ -z "$1" ]; then
    echo "❌ 请提供新项目名称"
    echo "用法: ./scripts/init.sh <新项目名>"
    echo "示例: ./scripts/init.sh my-awesome-app"
    exit 1
fi

NEW_NAME="$1"
OLD_NAME="spark"

echo "🚀 初始化项目: $NEW_NAME"
echo "================================"

# 检查是否在项目根目录
if [ ! -f "go.mod" ] || [ ! -d "web" ]; then
    echo "❌ 请在项目根目录运行此脚本"
    exit 1
fi

# 替换 Go 模块名
echo "📦 更新 Go 模块名..."
sed -i.bak "s|$OLD_NAME|$NEW_NAME|g" go.mod
rm -f go.mod.bak

# 替换所有 Go 文件中的导入路径
echo "🔧 更新 Go 导入路径..."
find . -name "*.go" -type f -exec sed -i.bak "s|$OLD_NAME|$NEW_NAME|g" {} +
find . -name "*.bak" -type f -delete

# 替换 proto 文件
echo "📝 更新 Proto 文件..."
find . -name "*.proto" -type f -exec sed -i.bak "s|$OLD_NAME|$NEW_NAME|g" {} +
find . -name "*.bak" -type f -delete

# 替换前端 package.json
echo "🎨 更新前端配置..."
sed -i.bak "s|\"name\": \"web\"|\"name\": \"$NEW_NAME-frontend\"|g" web/package.json
rm -f web/package.json.bak

# 替换 Docker 配置
echo "🐳 更新 Docker 配置..."
find docker -name "*.yml" -o -name "*.yaml" -o -name "Dockerfile" | while read f; do
    sed -i.bak "s|$OLD_NAME|$NEW_NAME|g" "$f"
    rm -f "$f.bak"
done

# 替换 Makefile
echo "📋 更新 Makefile..."
sed -i.bak "s|$OLD_NAME|$NEW_NAME|g" Makefile
rm -f Makefile.bak

# 替换 README
echo "📚 更新 README..."
sed -i.bak "s|$OLD_NAME|$NEW_NAME|g" README.md
rm -f README.md.bak

# 替换配置文件
echo "⚙️  更新配置文件..."
find . -name "*.yaml" -o -name "*.yml" -o -name "*.json" | grep -v node_modules | grep -v vendor | while read f; do
    sed -i.bak "s|$OLD_NAME|$NEW_NAME|g" "$f"
    rm -f "$f.bak"
done

echo ""
echo "✅ 项目初始化完成!"
echo ""
echo "下一步:"
echo "  1. 检查 go.mod 中的模块名是否正确"
echo "  2. 修改 docker/backend/config/config.yaml 配置"
echo "  3. 运行 make infra-up 启动基础设施"
echo "  4. 运行 make migrate-dev 初始化数据库"
echo "  5. 运行 make run 启动后端"
echo "  6. cd web && pnpm install && pnpm dev 启动前端"
echo ""
echo "🎉 开始开发吧!"
