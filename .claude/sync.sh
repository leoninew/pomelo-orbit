#!/usr/bin/env bash

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SOURCE_DIR="$SCRIPT_DIR/skills"
CLAUDE_TARGET="$HOME/.claude/skills"
KIRO_TARGET="$HOME/.kiro/skills"

show_help() {
    cat << 'EOF'
sync.sh - 同步 skills 目录与 ~/.claude/skills 和 ~/.kiro/skills

用法:
    ./sync.sh send     将本地 skills 同步到 claude 和 kiro
    ./sync.sh fetch    将 ~/.claude/skills 同步到本地
    ./sync.sh -h       显示此帮助

说明:
    send  - 本地 -> ~/.claude/skills 和 ~/.kiro/skills
    fetch - ~/.claude/skills -> 本地
EOF
}

if [[ $# -eq 0 || "$1" == "-h" || "$1" == "--help" ]]; then
    show_help
    exit 0
fi

if [[ "$1" == "send" ]]; then
    echo "[INFO] 本地 -> ~/.claude/skills"
    mkdir -p "$CLAUDE_TARGET"
    rsync -av "$SOURCE_DIR/" "$CLAUDE_TARGET/"

    echo "[INFO] 本地 -> ~/.kiro/skills"
    mkdir -p "$KIRO_TARGET"
    rsync -av "$SOURCE_DIR/" "$KIRO_TARGET/"

    echo "[INFO] 同步完成"
elif [[ "$1" == "fetch" ]]; then
    echo "[INFO] ~/.claude/skills -> 本地"
    mkdir -p "$SOURCE_DIR"
    rsync -av "$CLAUDE_TARGET/" "$SOURCE_DIR/"
    echo "[INFO] 同步完成"
else
    echo "[ERROR] 未知命令: $1"
    show_help
    exit 1
fi
