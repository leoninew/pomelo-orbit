.PHONY: help install dev-backend dev-frontend lint test test-backend test-frontend build clean

help:
	@echo "Pomelo Orbit - 开发命令"
	@echo ""
	@echo "安装依赖:"
	@echo "  make install         - 安装前后端所有依赖"
	@echo "  make install force=1 - 强制重新安装所有依赖"
	@echo ""
	@echo "开发服务:"
	@echo "  make dev-backend     - 启动后端服务器 (端口 9001)"
	@echo "  make dev-frontend    - 启动前端服务器 (端口 9002)"
	@echo ""
	@echo "代码检查:"
	@echo "  make lint            - 检查代码问题"
	@echo "  make lint fix=1      - 检查并自动修复"
	@echo ""
	@echo "测试:"
	@echo "  make test            - 运行所有测试（后端+前端，含集成测试）"
	@echo "  make test-backend    - 运行后端所有测试（默认含集成测试）"
	@echo "  make test-backend integration=0 - 仅运行后端单元测试（跳过集成测试）"
	@echo "  make test-backend cov=1 - 运行后端所有测试并生成覆盖率报告"
	@echo "  make test-backend integration=0 cov=1 - 仅运行单元测试并生成覆盖率报告"
	@echo "  make test-frontend   - 运行前端测试"
	@echo "  make test-frontend cov=1 - 运行前端测试并生成覆盖率报告"
	@echo ""
	@echo "构建:"
	@echo "  make build           - 构建 Docker 镜像 (默认 tag: latest)"
	@echo "  make build tag=v1.0  - 构建指定 tag 的镜像"
	@echo "  make build cn=1      - 使用国内镜像源构建"
	@echo ""
	@echo "清理:"
	@echo "  make clean           - 删除编译缓存目录（__pycache__、.mypy_cache、.ruff_cache、.pytest_cache、htmlcov）"
	@echo ""

install:
	@echo "正在安装后端依赖..."
	$(if $(force), \
		cd backend && UV_LINK_MODE=copy uv sync --all-groups --force-reinstall, \
		cd backend && UV_LINK_MODE=copy uv sync --all-groups)
	@echo "正在安装前端依赖..."
	$(if $(force), \
		cd frontend && yarn install --force, \
		cd frontend && yarn install)
	@echo "✓ 所有依赖安装完成"

dev-backend:
	@echo "启动后端服务器 (端口 9001)..."
	cd backend && PYTHONUTF8=1 uv run python -m pomelo_orbit.main --port 9001 --reload

dev-frontend:
	@echo "启动前端服务器 (端口 9002)..."
	cd frontend && yarn dev

lint:
	@echo "后端代码检查..."
	$(if $(fix), \
		cd backend && uv run ruff check --fix . && uv run ruff format ., \
		cd backend && uv run ruff check .)
	cd backend && PYTHONUTF8=1 uv run mypy src/ tests/
	@echo "前端代码检查..."
	$(if $(fix), \
		cd frontend && yarn format && yarn lint:fix, \
		cd frontend && yarn lint)

test: test-backend test-frontend
	@echo "✓ 所有测试完成"

test-backend:
	@echo "运行后端测试..."
	$(if $(cov), \
		$(if $(filter 0,$(integration)), \
			cd backend && PYTHONUTF8=1 uv run pytest tests/unit/ --cov=src/pomelo_orbit --cov-report=html --cov-report=term && echo "" && echo "✓ 覆盖率报告: backend/htmlcov/index.html", \
			cd backend && PYTHONUTF8=1 uv run pytest --cov=src/pomelo_orbit --cov-report=html --cov-report=term && echo "" && echo "✓ 覆盖率报告: backend/htmlcov/index.html"), \
		$(if $(filter 0,$(integration)), \
			cd backend && PYTHONUTF8=1 uv run pytest tests/unit/, \
			cd backend && PYTHONUTF8=1 uv run pytest))

test-frontend:
	@echo "运行前端测试..."
	$(if $(cov), \
		cd frontend && yarn test:cov && echo "" && echo "✓ 覆盖率报告: frontend/coverage/index.html", \
		cd frontend && yarn test)

build:
	@echo "构建 Docker 镜像..."
	docker build -f $(if $(cn),Dockerfile.cn,Dockerfile) -t pomelo-orbit:$(or $(tag),latest) .
	@echo "✓ 镜像构建完成: pomelo-orbit:$(or $(tag),latest)"

clean:
	@echo "清理编译缓存..."
	find . -type d -name "__pycache__" -not -path "./.git/*" | xargs rm -rf
	find . -type d -name ".mypy_cache" -not -path "./.git/*" | xargs rm -rf
	find . -type d -name ".ruff_cache" -not -path "./.git/*" | xargs rm -rf
	find . -type d -name ".pytest_cache" -not -path "./.git/*" | xargs rm -rf
	find . -type d -name "htmlcov" -not -path "./.git/*" | xargs rm -rf
	find . -type f -name "*.pyc" -not -path "./.git/*" | xargs rm -f
	find .
	@echo "✓ 清理完成"

