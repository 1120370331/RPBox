#!/bin/bash

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
MEMORIES_DIR="$(dirname "$SCRIPT_DIR")"

echo ""
echo "========================================"
echo "  RPBox .memories 速查工具"
echo "========================================"
echo ""

show_menu() {
    echo "[1] 查看模块索引"
    echo "[2] 列出所有模块"
    echo "[3] 搜索关键词"
    echo "[4] 打开模板目录"
    echo "[0] 退出"
    echo ""
}

while true; do
    show_menu
    read -p "请选择: " choice

    case $choice in
        1)
            echo ""
            cat "$MEMORIES_DIR/modules/INDEX.md"
            echo ""
            ;;
        2)
            echo ""
            echo "模块列表:"
            ls -d "$MEMORIES_DIR/modules"/*/ 2>/dev/null | xargs -n1 basename
            echo ""
            ;;
        3)
            read -p "输入关键词: " keyword
            echo ""
            grep -rni "$keyword" "$MEMORIES_DIR" --include="*.md"
            echo ""
            ;;
        4)
            if command -v open &> /dev/null; then
                open "$MEMORIES_DIR/templates"
            elif command -v xdg-open &> /dev/null; then
                xdg-open "$MEMORIES_DIR/templates"
            fi
            ;;
        0)
            exit 0
            ;;
        *)
            echo "无效选择"
            ;;
    esac
done
