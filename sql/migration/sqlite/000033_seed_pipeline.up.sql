-- pipeline seed captured from data/mysql-transfer-20260816-103803.mysql.sql.

-- repository: 1 row(s).
INSERT INTO "repository" ("id", "project_id", "name", "code", "repository_type", "repository_url", "git_credential_id", "variable_overrides", "default_branch") VALUES
    ('01M327332NTE0VY4S5YRWJ2HZR', '01KRRKK0K3T519ZQZES3M4QA9Z', 'Go Docker', 'go-docker', 'remote_git', 'https://github.com/callicoder/go-docker.git', NULL, '[]', 'master');

-- pipeline: 3 row(s).
INSERT INTO "pipeline" ("id", "project_id", "kind", "source_pipeline_id", "source_template_name", "source_template_version", "application_id", "application_name", "repository_id", "repository_name", "version_fork_strategy", "fixed_version_id", "fixed_version_label", "name", "description", "variable_declarations", "version") VALUES
    ('01M391Y93NTCXQ6J8H34VBMJJR', NULL, 'template', NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, '镜像构建流水线', '适用于使用 Dockerfile 进制镜像构建的仓库', '[{"default":null,"editable":true,"name":"image_name","secret":false,"source":"pipeline_custom","stage_id":"01M38WMQDCY38G5B09PVB697C5","value":"{{ repository_code }}","description":""}]', 13),
    ('01M391Y93NTCXQ6J8H38DXEN8Q', '01KRRKK0K3T519ZQZES3M4QA9Z', 'application', '01M391Y93NTCXQ6J8H34VBMJJR', '镜像构建流水线', 13, NULL, NULL, '01M327332NTE0VY4S5YRWJ2HZR', 'Go Docker', NULL, NULL, NULL, '镜像构建流水线： Go Docker', '适用于使用 Dockerfile 进制镜像构建的仓库', '[{"default":null,"description":"","editable":true,"name":"image_name","secret":false,"source":"pipeline_custom","stage_id":"01M38WMQDCY38G5B09PRXSWZXT","value":"{{ repository_code }}"}]', 2),
    ('01M391Y93PEYM4CNTGM5EBRS40', NULL, 'template', NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, 'Go 构建流水线', '- test & lint
- build', '[{"default":null,"editable":true,"name":"working_dir","secret":false,"source":"pipeline_custom","value":".","description":""}]', 12);

-- pipeline_stage: 7 row(s).
INSERT INTO "pipeline_stage" ("id", "project_id", "kind", "pipeline_id", "name", "image", "script", "description", "version", "source_template_stage_id", "source_template_stage_name", "source_template_stage_version", "source_template_stage_description", "artifacts", "depends_on", "sort_order") VALUES
    ('01M38WMQDCY38G5B09PD7JG1YW', NULL, 'template', NULL, 'git clone', 'alpine/git', 'set -e
# 将目录所有权改为当前执行用户
chown -R $(id -u):$(id -g) /workspace

# 强制全局关闭 SSL 验证
git config --global http.sslVerify "false"
git config --system http.sslVerify "false"
# 信任工作区目录
git config --global --add safe.directory /workspace
# 初始化仓库
git init
git remote remove origin 2>/dev/null || true
git remote add origin {{ repository_url }}
# 拉取代码
git fetch --depth=1 --force origin {{ repository_ref }}
git clean -fd
git checkout --force -B {{ repository_ref }} FETCH_HEAD', 'Clone source repository', 2, NULL, NULL, NULL, NULL, '[{"collector":"command","command":"git rev-parse HEAD","format":"git_object_id","name":"source_commit"}]', NULL, 1),
    ('01M38WMQDCY38G5B09PF8M85XG', NULL, 'template', NULL, 'docker build', 'docker:29.4', 'set -e

cd {{ working_dir | default: "." }}

# shallow clone 下不要用 build-version.sh（几乎只会得到 dev-1-g...）
COMMIT="$(git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)"
BUILD_TIME="{{ runtime_datetime }}"
# 放弃语义 version：与镜像 tag 对齐即可（或固定 dev）
VERSION="{{ runtime_datetime }}"

docker build -f {{ repository_dockerfile | default: "Dockerfile" }} \
  -t {{ image_name }}:{{ runtime_datetime }} \
  --build-arg VERSION="${VERSION}" \
  --build-arg COMMIT="${COMMIT}" \
  --build-arg BUILD_TIME="${BUILD_TIME}" \
  .', 'Build container image', 4, NULL, NULL, NULL, NULL, '[{"collector":"docker_image","reference":"{{ image_name }}:{{ runtime_datetime }}","name":"image"}]', NULL, 2),
    ('01M38WMQDCY38G5B09PGA57NVR', NULL, 'template', NULL, 'golang:1.25 lint', 'golang:1.25-alpine', 'set -e
cd {{ working_dir }}
go env -w GOPROXY=https://goproxy.cn,direct
go install golang.org/x/lint/golint@latest
golint ./...', '运行 Go 代码质量', 4, NULL, NULL, NULL, NULL, '[]', NULL, 3),
    ('01M38WMQDCY38G5B09PM30KXV1', NULL, 'template', NULL, 'golang:1.25 test', 'golang:1.25-alpine', 'set -e
cd {{ working_dir }}
go env -w GOPROXY=https://goproxy.cn,direct
go test ./...', '运行 Go 单元测试', 4, NULL, NULL, NULL, NULL, '[]', NULL, 4),
    ('01M38WMQDCY38G5B09PNFD2Q02', NULL, 'template', NULL, 'golang:1.25 build', 'golang:1.25-alpine', 'set -e
cd {{ working_dir }}
mkdir -p dist
go env -w GOPROXY=https://goproxy.cn,direct
go build -o dist/', '运行 Go 构建', 6, NULL, NULL, NULL, NULL, '[]', NULL, 5),
    ('01M38WMQDCY38G5B09PQ7QYWMA', '01KRRKK0K3T519ZQZES3M4QA9Z', 'application', '01M391Y93NTCXQ6J8H38DXEN8Q', 'git clone', 'alpine/git', 'set -e
# 将目录所有权改为当前执行用户
chown -R $(id -u):$(id -g) /workspace

# 强制全局关闭 SSL 验证
git config --global http.sslVerify "false"
git config --system http.sslVerify "false"
# 信任工作区目录
git config --global --add safe.directory /workspace
# 初始化仓库
git init
git remote remove origin 2>/dev/null || true
git remote add origin {{ repository_url }}
# 拉取代码
git fetch --depth=1 --force origin {{ repository_ref }}
git clean -fd
git checkout --force -B {{ repository_ref }} FETCH_HEAD', 'Clone source repository', NULL, '01M38WMQDCY38G5B09PD7JG1YW', 'git clone', 2, 'Clone source repository', '[{"collector":"command","command":"git rev-parse HEAD","format":"git_object_id","name":"source_commit"}]', '[]', 1),
    ('01M38WMQDCY38G5B09PRXSWZXT', '01KRRKK0K3T519ZQZES3M4QA9Z', 'application', '01M391Y93NTCXQ6J8H38DXEN8Q', 'docker build', 'docker:29.4', 'set -e
cd {{ working_dir | default: "." }}
docker build -t {{ image_name }}:{{ runtime_datetime }} -f {{ repository_dockerfile | default: "Dockerfile" }} .', 'Build container image', NULL, '01M38WMQDCY38G5B09PF8M85XG', 'docker build', 4, 'Build container image', '[{"collector":"docker_image","reference":"{{ image_name }}:{{ runtime_datetime }}","name":"image"}]', '["01M38WMQDCY38G5B09PQ7QYWMA"]', 2);

-- pipeline_stage_reference: 6 row(s).
INSERT INTO "pipeline_stage_reference" ("id", "pipeline_id", "source_template_stage_id", "source_template_stage_name", "source_template_stage_version", "source_template_stage_description", "name", "image", "script", "description", "artifacts", "depends_on", "sort_order") VALUES
    ('01M38WMQDCY38G5B09PTZ4KKAB', '01M391Y93NTCXQ6J8H34VBMJJR', '01M38WMQDCY38G5B09PD7JG1YW', 'git clone', 2, 'Clone source repository', 'git clone', 'alpine/git', 'set -e
# 将目录所有权改为当前执行用户
chown -R $(id -u):$(id -g) /workspace

# 强制全局关闭 SSL 验证
git config --global http.sslVerify "false"
git config --system http.sslVerify "false"
# 信任工作区目录
git config --global --add safe.directory /workspace
# 初始化仓库
git init
git remote remove origin 2>/dev/null || true
git remote add origin {{ repository_url }}
# 拉取代码
git fetch --depth=1 --force origin {{ repository_ref }}
git clean -fd
git checkout --force -B {{ repository_ref }} FETCH_HEAD', 'Clone source repository', '[{"collector":"command","command":"git rev-parse HEAD","format":"git_object_id","name":"source_commit"}]', '[]', 1),
    ('01M38WMQDCY38G5B09PVB697C5', '01M391Y93NTCXQ6J8H34VBMJJR', '01M38WMQDCY38G5B09PF8M85XG', 'docker build', 4, 'Build container image', 'docker build', 'docker:29.4', 'set -e
cd {{ working_dir | default: "." }}
docker build -t {{ image_name }}:{{ runtime_datetime }} -f {{ repository_dockerfile | default: "Dockerfile" }} .', 'Build container image', '[{"collector":"docker_image","reference":"{{ image_name }}:{{ runtime_datetime }}","name":"image"}]', '["01M38WMQDCY38G5B09PTZ4KKAB"]', 2),
    ('01M38WMQDCY38G5B09PX20TZ25', '01M391Y93PEYM4CNTGM5EBRS40', '01M38WMQDCY38G5B09PD7JG1YW', 'git clone (backup VJGR9B)', 8, '克隆代码仓库', 'git clone', 'alpine/git', 'set -e
# 将目录所有权改为当前执行用户
chown -R $(id -u):$(id -g) /workspace

# 强制全局关闭 SSL 验证
git config --global http.sslVerify "false"
git config --system http.sslVerify "false"
# 信任工作区目录
git config --global --add safe.directory /workspace
# 初始化仓库
git init
git remote remove origin 2>/dev/null || true
git remote add origin {{ repository_url }}
# 拉取代码
git fetch --depth=1 --force origin {{ repository_ref }}
git clean -fd
git checkout --force -B {{ repository_ref }} FETCH_HEAD', '克隆代码仓库', '[]', '[]', 1),
    ('01M38WMQDCY38G5B09Q0AV391Y', '01M391Y93PEYM4CNTGM5EBRS40', '01M38WMQDCY38G5B09PGA57NVR', 'golang:1.23 lint', 3, '运行 Go 代码质量', 'golang:1.25 lint', 'golang:1.23-alpine', 'set -e
cd {{ working_dir }}
go env -w GOPROXY=https://goproxy.cn,direct
go install golang.org/x/lint/golint@latest
golint ./...', '运行 Go 代码质量', '[]', '["01M38WMQDCY38G5B09PX20TZ25"]', 2),
    ('01M38WMQDCY38G5B09Q43VEYT7', '01M391Y93PEYM4CNTGM5EBRS40', '01M38WMQDCY38G5B09PM30KXV1', 'golang:1.23 test', 3, '运行 Go 单元测试', 'golang:1.25 test', 'golang:1.23-alpine', 'set -e
cd {{ working_dir }}
go env -w GOPROXY=https://goproxy.cn,direct
go test ./...', '运行 Go 单元测试', '[]', '["01M38WMQDCY38G5B09PX20TZ25"]', 3),
    ('01M38WMQDCY38G5B09Q4NAGD3M', '01M391Y93PEYM4CNTGM5EBRS40', '01M38WMQDCY38G5B09PNFD2Q02', 'golang:1.23 build', 5, '运行 Go 构建', 'golang:1.25 build', 'golang:1.23-alpine', 'set -e
cd {{ working_dir }}
mkdir -p dist
go env -w GOPROXY=https://goproxy.cn,direct
go build -o dist/', '运行 Go 构建', '[]', '["01M38WMQDCY38G5B09Q43VEYT7"]', 4);
