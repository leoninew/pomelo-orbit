-- pipeline seed captured from data/mysql-transfer-20260816-103803.mysql.sql.
-- Upserts support databases previously initialized by the removed data-migration loader.

-- pipeline: 2 row(s).
INSERT INTO "pipeline" ("id", "project_id", "kind", "source_pipeline_id", "source_template_name", "source_template_version", "application_id", "application_name", "repository_id", "repository_name", "version_fork_strategy", "fixed_version_id", "fixed_version_label", "name", "description", "variable_declarations", "version") VALUES
    ('01KNVEJPWVK757139NMNNNCEFE', '01KRRKK0K3T519ZQZES3M4QA9Z', 'template', NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, 'Go 构建流水线', '- test & lint
- build', '[{"default":null,"editable":true,"name":"working_dir","secret":false,"source":"pipeline_custom","value":".","description":""}]', 12),
    ('01KZG83K2MXG08EJ6G48SG38B3', '01KRRKK0K3T519ZQZES3M4QA9Z', 'template', NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, NULL, '镜像构建流水线', '适用于使用 Dockerfile 进制镜像构建的仓库', '[]', 10)
ON CONFLICT DO UPDATE SET
    "project_id" = excluded."project_id",
    "kind" = excluded."kind",
    "source_pipeline_id" = excluded."source_pipeline_id",
    "source_template_name" = excluded."source_template_name",
    "source_template_version" = excluded."source_template_version",
    "application_id" = excluded."application_id",
    "application_name" = excluded."application_name",
    "repository_id" = excluded."repository_id",
    "repository_name" = excluded."repository_name",
    "version_fork_strategy" = excluded."version_fork_strategy",
    "fixed_version_id" = excluded."fixed_version_id",
    "fixed_version_label" = excluded."fixed_version_label",
    "name" = excluded."name",
    "description" = excluded."description",
    "variable_declarations" = excluded."variable_declarations",
    "version" = excluded."version";

-- pipeline_stage: 5 row(s).
INSERT INTO "pipeline_stage" ("id", "project_id", "kind", "pipeline_id", "name", "image", "script", "description", "version", "source_template_stage_id", "source_template_stage_name", "source_template_stage_version", "source_template_stage_description", "artifacts", "depends_on", "sort_order") VALUES
    ('01KNRANZDR4PASATAXKTBBTRX9', '01KRRKK0K3T519ZQZES3M4QA9Z', 'template', NULL, 'golang:1.25 test', 'golang:1.25-alpine', 'set -e
cd {{ working_dir }}
go env -w GOPROXY=https://goproxy.cn,direct
go test ./...', '运行 Go 单元测试', 4, NULL, NULL, NULL, NULL, '[]', NULL, NULL),
    ('01KNRDSSJ7RNND7110175N4NR2', '01KRRKK0K3T519ZQZES3M4QA9Z', 'template', NULL, 'golang:1.25 build', 'golang:1.25-alpine', 'set -e
cd {{ working_dir }}
mkdir -p dist
go env -w GOPROXY=https://goproxy.cn,direct
go build -o dist/', '运行 Go 构建', 6, NULL, NULL, NULL, NULL, '[]', NULL, NULL),
    ('01KNRKNAHG3EBS07VBK2YY5ZQN', '01KRRKK0K3T519ZQZES3M4QA9Z', 'template', NULL, 'golang:1.25 lint', 'golang:1.25-alpine', 'set -e
cd {{ working_dir }}
go env -w GOPROXY=https://goproxy.cn,direct
go install golang.org/x/lint/golint@latest
golint ./...', '运行 Go 代码质量', 4, NULL, NULL, NULL, NULL, '[]', NULL, NULL),
    ('01KRCWNJVA1DM02TJXZ4STJD01', '01KRRKK0K3T519ZQZES3M4QA9Z', 'template', NULL, 'git clone', 'alpine/git', 'set -e
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
git fetch --depth=1 origin {{ repository_ref }}
git checkout -B {{ repository_ref }} FETCH_HEAD', 'Clone source repository', 2, NULL, NULL, NULL, NULL, '[{"collector":"command","command":"git rev-parse HEAD","format":"git_object_id","name":"source_commit"}]', NULL, NULL),
    ('01KRCWNJVA1DM02TJXZ4STJD06', '01KRRKK0K3T519ZQZES3M4QA9Z', 'template', NULL, 'docker build', 'docker:29.4', 'set -e

cd {{ working_dir }}

# shallow clone 下不要用 build-version.sh（几乎只会得到 dev-1-g...）
COMMIT="$(git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)"
BUILD_TIME="{{ runtime_datetime }}"
# 放弃语义 version：与镜像 tag 对齐即可（或固定 dev）
VERSION="{{ runtime_datetime }}"

docker build -f {{ repository_dockerfile }} \
  -t {{ repository_code }}:{{ runtime_datetime }} \
  --build-arg VERSION="${VERSION}" \
  --build-arg COMMIT="${COMMIT}" \
  --build-arg BUILD_TIME="${BUILD_TIME}" \
  .', 'Build container image', 2, NULL, NULL, NULL, NULL, '[{"collector":"docker_image","reference":"{{ repository_code }}:{{ runtime_datetime }}","name":"{{ repository_code }}"}]', NULL, NULL)
ON CONFLICT DO UPDATE SET
    "project_id" = excluded."project_id",
    "kind" = excluded."kind",
    "pipeline_id" = excluded."pipeline_id",
    "name" = excluded."name",
    "image" = excluded."image",
    "script" = excluded."script",
    "description" = excluded."description",
    "version" = excluded."version",
    "source_template_stage_id" = excluded."source_template_stage_id",
    "source_template_stage_name" = excluded."source_template_stage_name",
    "source_template_stage_version" = excluded."source_template_stage_version",
    "source_template_stage_description" = excluded."source_template_stage_description",
    "artifacts" = excluded."artifacts",
    "depends_on" = excluded."depends_on",
    "sort_order" = excluded."sort_order";

-- pipeline_stage_reference: 6 row(s).
INSERT INTO "pipeline_stage_reference" ("id", "pipeline_id", "source_template_stage_id", "source_template_stage_name", "source_template_stage_version", "source_template_stage_description", "name", "image", "script", "description", "artifacts", "depends_on", "sort_order") VALUES
    ('01KZGBG9NT6NCK8AT6H6ENV874', '01KZG83K2MXG08EJ6G48SG38B3', '01KRCWNJVA1DM02TJXZ4STJD01', 'git clone', 2, 'Clone source repository', 'git clone', 'alpine/git', 'set -e
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
git fetch --depth=1 origin {{ repository_ref }}
git checkout -B {{ repository_ref }} FETCH_HEAD', 'Clone source repository', '[{"collector":"command","command":"git rev-parse HEAD","format":"git_object_id","name":"source_commit"}]', '[]', 1),
    ('01KZGBGDK1G249681EVBDA9035', '01KZG83K2MXG08EJ6G48SG38B3', '01KRCWNJVA1DM02TJXZ4STJD06', 'docker build', 1, 'Build container image', 'docker build', 'docker:29.4', 'set -e
cd {{ working_dir }}
docker build -t {{ repository_code }}:{{ runtime_datetime }} -f {{ repository_dockerfile }} .', 'Build container image', '[{"collector":"docker_image","reference":"{{ repository_code }}:{{ runtime_datetime }}","name":"{{ repository_code }}"}]', '["01KZGBG9NT6NCK8AT6H6ENV874"]', 2),
    ('4smvyi2oq4n2kzrio4zmcqykge', '01KNVEJPWVK757139NMNNNCEFE', '01KNRANZDR4PASATAXKTBBTRX9', 'golang:1.23 test', 3, '运行 Go 单元测试', 'golang:1.25 test', 'golang:1.23-alpine', 'set -e
cd {{ working_dir }}
go env -w GOPROXY=https://goproxy.cn,direct
go test ./...', '运行 Go 单元测试', '[]', '["vb37nbzugq6pljhkbxm3hiij24"]', 1),
    ('cirr32qhvah2rbv2obb76f7fte', '01KNVEJPWVK757139NMNNNCEFE', '01KNRKNAHG3EBS07VBK2YY5ZQN', 'golang:1.23 lint', 3, '运行 Go 代码质量', 'golang:1.25 lint', 'golang:1.23-alpine', 'set -e
cd {{ working_dir }}
go env -w GOPROXY=https://goproxy.cn,direct
go install golang.org/x/lint/golint@latest
golint ./...', '运行 Go 代码质量', '[]', '["vb37nbzugq6pljhkbxm3hiij24"]', 2),
    ('vb37nbzugq6pljhkbxm3hiij24', '01KNVEJPWVK757139NMNNNCEFE', '01KZ5A17696GZ6NS5BS6VJGR9B', 'git clone (backup VJGR9B)', 8, '克隆代码仓库', 'git clone', 'alpine/git', 'set -e
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
git fetch --depth=1 origin {{ repository_ref }}
git checkout -B {{ repository_ref }} FETCH_HEAD', '克隆代码仓库', '[]', '[]', 0),
    ('yrkdm4fc4wlupvd3ne6ea2ywpe', '01KNVEJPWVK757139NMNNNCEFE', '01KNRDSSJ7RNND7110175N4NR2', 'golang:1.23 build', 5, '运行 Go 构建', 'golang:1.25 build', 'golang:1.23-alpine', 'set -e
cd {{ working_dir }}
mkdir -p dist
go env -w GOPROXY=https://goproxy.cn,direct
go build -o dist/', '运行 Go 构建', '[]', '["4smvyi2oq4n2kzrio4zmcqykge"]', 3)
ON CONFLICT DO UPDATE SET
    "pipeline_id" = excluded."pipeline_id",
    "source_template_stage_id" = excluded."source_template_stage_id",
    "source_template_stage_name" = excluded."source_template_stage_name",
    "source_template_stage_version" = excluded."source_template_stage_version",
    "source_template_stage_description" = excluded."source_template_stage_description",
    "name" = excluded."name",
    "image" = excluded."image",
    "script" = excluded."script",
    "description" = excluded."description",
    "artifacts" = excluded."artifacts",
    "depends_on" = excluded."depends_on",
    "sort_order" = excluded."sort_order";
