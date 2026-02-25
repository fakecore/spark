#!/bin/bash

# 项目模板初始化脚本
# 用法: curl -fsSL https://raw.githubusercontent.com/fakecore/project-template/main/scripts/setup.sh | bash -s my-project

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 打印带颜色的消息
info() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# 检查参数
PROJECT_NAME="${1:-}"
if [ -z "$PROJECT_NAME" ]; then
    error "请提供项目名称\n用法: curl -fsSL <url> | bash -s <项目名>\n示例: curl -fsSL <url> | bash -s my-awesome-app"
fi

# 检查必要工具
check_deps() {
    local deps=("git" "sed")
    for dep in "${deps[@]}"; do
        if ! command -v "$dep" &> /dev/null; then
            error "缺少必要工具: $dep"
        fi
    done
}

# 克隆模板仓库
clone_template() {
    info "克隆模板仓库..."
    if [ -d "$PROJECT_NAME" ]; then
        error "目录 $PROJECT_NAME 已存在"
    fi
    git clone --depth 1 https://github.com/fakecore/spark.git "$PROJECT_NAME"
    cd "$PROJECT_NAME"
    rm -rf .git
    git init
    git add .
    git commit -m "Initial commit from template"
}

# 替换项目名称
replace_names() {
    info "替换项目名称..."
    local old_name="project-template"

    # Go 模块
    sed -i.bak "s|$old_name|$PROJECT_NAME|g" go.mod
    rm -f go.mod.bak

    # Go 文件
    find . -name "*.go" -type f -exec sed -i.bak "s|$old_name|$PROJECT_NAME|g" {} +
    find . -name "*.bak" -type f -delete

    # Proto 文件
    find . -name "*.proto" -type f -exec sed -i.bak "s|$old_name|$PROJECT_NAME|g" {} +
    find . -name "*.bak" -type f -delete

    # 前端
    sed -i.bak "s|\"name\": \"web\"|\"name\": \"$PROJECT_NAME-frontend\"|g" web/package.json
    rm -f web/package.json.bak

    # Docker
    find docker -name "*.yml" -o -name "*.yaml" -o -name "Dockerfile" 2>/dev/null | while read f; do
        sed -i.bak "s|$old_name|$PROJECT_NAME|g" "$f"
        rm -f "$f.bak"
    done

    # Makefile
    sed -i.bak "s|$old_name|$PROJECT_NAME|g" Makefile
    rm -f Makefile.bak

    # 配置文件
    find . -name "*.yaml" -o -name "*.yml" -o -name "*.json" | grep -v node_modules | grep -v vendor | while read f; do
        sed -i.bak "s|$old_name|$PROJECT_NAME|g" "$f"
        rm -f "$f.bak"
    done
}

# 清理模板文件
cleanup() {
    info "清理模板文件..."
    rm -rf scripts/setup.sh
    rm -rf .github/workflows/template-*.yml 2>/dev/null || true
}

# 主流程
main() {
    echo ""
    echo "🚀 项目模板初始化"
    echo "=================="
    echo "项目名称: $PROJECT_NAME"
    echo ""

    check_deps
    clone_template
    replace_names
    cleanup

    echo ""
    echo -e "${GREEN}✅ 项目初始化完成!${NC}"
    echo ""
    echo "下一步:"
    echo "  cd $PROJECT_NAME"
    echo "  vim docker/backend/config/config.yaml  # 修改配置"
    echo "  make infra-up                           # 启动基础设施"
    echo "  make migrate-dev                        # 初始化数据库"
    echo "  make run                                # 启动后端"
    echo "  cd web && pnpm install && pnpm dev      # 启动前端"
    echo ""
    echo "🎉 开始开发吧!"
}

main
