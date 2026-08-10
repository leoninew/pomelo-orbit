-- Business data migration: default-project reusable pipeline templates.
--
-- The rows below are synchronized from the development template library. These
-- are managed defaults, so rerunning this data migration refreshes records with
-- the same stable identity instead of retaining stale seed content.

INSERT INTO pipeline_stage (
    id, project_id, kind, name, image, script, description, version, artifacts, created_at, updated_at
) VALUES
    (
        '01KNRANZDR4PASATAXKTBBTRX9',
        '01KRRKK0K3T519ZQZES3M4QA9Z',
        'template',
        'golang:1.25 test',
        'golang:1.25-alpine',
        'set -e' || char(10) ||
        'cd {{ working_dir }}' || char(10) ||
        'go env -w GOPROXY=https://goproxy.cn,direct' || char(10) ||
        'go test ./...',
        '运行 Go 单元测试',
        4,
        '[]',
        '2024-03-16T00:00:00Z',
        '2026-08-10 02:23:40.0606105 +0000 UTC'
    ),
    (
        '01KNRDSSJ7RNND7110175N4NR2',
        '01KRRKK0K3T519ZQZES3M4QA9Z',
        'template',
        'golang:1.25 build',
        'golang:1.25-alpine',
        'set -e' || char(10) ||
        'cd {{ working_dir }}' || char(10) ||
        'mkdir -p dist' || char(10) ||
        'go env -w GOPROXY=https://goproxy.cn,direct' || char(10) ||
        'go build -o dist/',
        '运行 Go 构建',
        6,
        '[]',
        '2024-03-16T00:00:00Z',
        '2026-08-10 02:23:48.813319 +0000 UTC'
    ),
    (
        '01KNRKNAHG3EBS07VBK2YY5ZQN',
        '01KRRKK0K3T519ZQZES3M4QA9Z',
        'template',
        'golang:1.25 lint',
        'golang:1.25-alpine',
        'set -e' || char(10) ||
        'cd {{ working_dir }}' || char(10) ||
        'go env -w GOPROXY=https://goproxy.cn,direct' || char(10) ||
        'go install golang.org/x/lint/golint@latest' || char(10) ||
        'golint ./...',
        '运行 Go 代码质量',
        4,
        '[]',
        '2024-03-16T00:00:00Z',
        '2026-08-10 02:23:55.9626765 +0000 UTC'
    ),
    (
        '01KRCWNJVA1DM02TJXZ4STJD01',
        '01KRRKK0K3T519ZQZES3M4QA9Z',
        'template',
        'git clone',
        'alpine/git',
        'set -e' || char(10) ||
        '# 强制全局关闭 SSL 验证' || char(10) ||
        'git config --global http.sslVerify "false"' || char(10) ||
        'git config --system http.sslVerify "false"' || char(10) ||
        '# 信任工作区目录' || char(10) ||
        'git config --global --add safe.directory /workspace' || char(10) ||
        '# 初始化仓库' || char(10) ||
        'git init' || char(10) ||
        'git remote remove origin 2>/dev/null || true' || char(10) ||
        'git remote add origin {{ repository_url }}' || char(10) ||
        '# 拉取代码' || char(10) ||
        'git fetch --depth=1 origin {{ repository_ref }}' || char(10) ||
        'git checkout -B {{ repository_ref }} FETCH_HEAD',
        'Clone source repository',
        2,
        '[{"collector":"command","command":"git rev-parse HEAD","format":"git_object_id","name":"source_commit"}]',
        '2024-03-16T00:00:00Z',
        '2026-08-10 02:20:55.9377211 +0000 UTC'
    ),
    (
        '01KRCWNJVA1DM02TJXZ4STJD06',
        '01KRRKK0K3T519ZQZES3M4QA9Z',
        'template',
        'docker build',
        'docker:29.4',
        'set -e' || char(10) || char(10) ||
        'cd {{ working_dir }}' || char(10) || char(10) ||
        '# shallow clone 下不要用 build-version.sh（几乎只会得到 dev-1-g...）' || char(10) ||
        'COMMIT="$(git rev-parse --short=12 HEAD 2>/dev/null || echo unknown)"' || char(10) ||
        'BUILD_TIME="{{ runtime_datetime }}"' || char(10) ||
        '# 放弃语义 version：与镜像 tag 对齐即可（或固定 dev）' || char(10) ||
        'VERSION="{{ runtime_datetime }}"' || char(10) || char(10) ||
        'docker build -f {{ repository_dockerfile }} \' || char(10) ||
        '  -t {{ repository_code }}:{{ runtime_datetime }} \' || char(10) ||
        '  --build-arg VERSION="${VERSION}" \' || char(10) ||
        '  --build-arg COMMIT="${COMMIT}" \' || char(10) ||
        '  --build-arg BUILD_TIME="${BUILD_TIME}" \' || char(10) ||
        '  .',
        'Build container image',
        2,
        '[{"collector":"docker_image","reference":"{{ repository_code }}:{{ runtime_datetime }}","name":"{{ repository_code }}"}]',
        '2024-03-16T00:00:00Z',
        '2026-08-10 02:22:09.6679008 +0000 UTC'
    )
ON CONFLICT(id) DO UPDATE SET
    project_id = excluded.project_id,
    kind = excluded.kind,
    pipeline_id = NULL,
    name = excluded.name,
    image = excluded.image,
    script = excluded.script,
    description = excluded.description,
    version = excluded.version,
    source_template_stage_id = NULL,
    source_template_stage_name = NULL,
    source_template_stage_version = NULL,
    source_template_stage_description = NULL,
    artifacts = excluded.artifacts,
    depends_on = NULL,
    sort_order = NULL,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at;

INSERT INTO pipeline (
    id, project_id, kind, name, description, variable_declarations, version, created_at, updated_at
) VALUES
    (
        '01KNVEJPWVK757139NMNNNCEFE',
        '01KRRKK0K3T519ZQZES3M4QA9Z',
        'template',
        'Go 构建流水线',
        '- test & lint' || char(10) || '- build',
        '[{"default":null,"editable":true,"name":"working_dir","secret":false,"source":"pipeline_custom","value":".","description":""}]',
        12,
        '2024-03-16T00:00:00Z',
        '2026-08-10 02:24:31.0997592 +0000 UTC'
    ),
    (
        '01KZG83K2MXG08EJ6G48SG38B3',
        '01KRRKK0K3T519ZQZES3M4QA9Z',
        'template',
        '镜像构建流水线',
        '适用于使用 Dockerfile 进制镜像构建的仓库',
        '[]',
        10,
        '2026-08-08T08:33:50.6767901Z',
        '2026-08-10 02:23:03.7371925 +0000 UTC'
    )
ON CONFLICT(id) DO UPDATE SET
    project_id = excluded.project_id,
    kind = excluded.kind,
    source_pipeline_id = NULL,
    source_template_name = NULL,
    source_template_version = NULL,
    application_id = NULL,
    application_name = NULL,
    repository_id = NULL,
    repository_name = NULL,
    version_fork_strategy = NULL,
    fixed_version_id = NULL,
    fixed_version_label = NULL,
    name = excluded.name,
    description = excluded.description,
    variable_declarations = excluded.variable_declarations,
    version = excluded.version,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at;

INSERT INTO pipeline_stage_reference (
    id,
    pipeline_id,
    source_template_stage_id,
    source_template_stage_name,
    source_template_stage_version,
    source_template_stage_description,
    name,
    image,
    script,
    description,
    artifacts,
    depends_on,
    sort_order,
    created_at,
    updated_at
) VALUES
    (
        '01KZGBG9NT6NCK8AT6H6ENV874',
        '01KZG83K2MXG08EJ6G48SG38B3',
        '01KRCWNJVA1DM02TJXZ4STJD01',
        'git clone',
        2,
        'Clone source repository',
        'git clone',
        'alpine/git',
        'set -e' || char(10) ||
        '# 强制全局关闭 SSL 验证' || char(10) ||
        'git config --global http.sslVerify "false"' || char(10) ||
        'git config --system http.sslVerify "false"' || char(10) ||
        '# 信任工作区目录' || char(10) ||
        'git config --global --add safe.directory /workspace' || char(10) ||
        '# 初始化仓库' || char(10) ||
        'git init' || char(10) ||
        'git remote remove origin 2>/dev/null || true' || char(10) ||
        'git remote add origin {{ repository_url }}' || char(10) ||
        '# 拉取代码' || char(10) ||
        'git fetch --depth=1 origin {{ repository_ref }}' || char(10) ||
        'git checkout -B {{ repository_ref }} FETCH_HEAD',
        'Clone source repository',
        '[{"collector":"command","command":"git rev-parse HEAD","format":"git_object_id","name":"source_commit"}]',
        '[]',
        1,
        '2026-08-08 09:33:12.7622645 +0000 UTC',
        '2026-08-08 09:33:12.7622645 +0000 UTC'
    ),
    (
        '01KZGBGDK1G249681EVBDA9035',
        '01KZG83K2MXG08EJ6G48SG38B3',
        '01KRCWNJVA1DM02TJXZ4STJD06',
        'docker build',
        1,
        'Build container image',
        'docker build',
        'docker:29.4',
        'set -e' || char(10) ||
        'cd {{ working_dir }}' || char(10) ||
        'docker build -t {{ repository_code }}:{{ runtime_datetime }} -f {{ repository_dockerfile }} .',
        'Build container image',
        '[{"collector":"docker_image","reference":"{{ repository_code }}:{{ runtime_datetime }}","name":"{{ repository_code }}"}]',
        '["01KZGBG9NT6NCK8AT6H6ENV874"]',
        2,
        '2026-08-08 09:33:16.7695648 +0000 UTC',
        '2026-08-08 09:33:16.7695648 +0000 UTC'
    ),
    (
        '4smvyi2oq4n2kzrio4zmcqykge',
        '01KNVEJPWVK757139NMNNNCEFE',
        '01KNRANZDR4PASATAXKTBBTRX9',
        'golang:1.23 test',
        3,
        '运行 Go 单元测试',
        'golang:1.25 test',
        'golang:1.23-alpine',
        'set -e' || char(10) ||
        'cd {{ working_dir }}' || char(10) ||
        'go env -w GOPROXY=https://goproxy.cn,direct' || char(10) ||
        'go test ./...',
        '运行 Go 单元测试',
        '[]',
        '["vb37nbzugq6pljhkbxm3hiij24"]',
        1,
        '2024-03-16 00:00:00 +0000 UTC',
        '2024-03-16 00:00:00 +0000 UTC'
    ),
    (
        'cirr32qhvah2rbv2obb76f7fte',
        '01KNVEJPWVK757139NMNNNCEFE',
        '01KNRKNAHG3EBS07VBK2YY5ZQN',
        'golang:1.23 lint',
        3,
        '运行 Go 代码质量',
        'golang:1.25 lint',
        'golang:1.23-alpine',
        'set -e' || char(10) ||
        'cd {{ working_dir }}' || char(10) ||
        'go env -w GOPROXY=https://goproxy.cn,direct' || char(10) ||
        'go install golang.org/x/lint/golint@latest' || char(10) ||
        'golint ./...',
        '运行 Go 代码质量',
        '[]',
        '["vb37nbzugq6pljhkbxm3hiij24"]',
        2,
        '2024-03-16 00:00:00 +0000 UTC',
        '2024-03-16 00:00:00 +0000 UTC'
    ),
    (
        'vb37nbzugq6pljhkbxm3hiij24',
        '01KNVEJPWVK757139NMNNNCEFE',
        '01KZ5A17696GZ6NS5BS6VJGR9B',
        'git clone (backup VJGR9B)',
        8,
        '克隆代码仓库',
        'git clone',
        'alpine/git',
        'set -e' || char(10) ||
        '# 强制全局关闭 SSL 验证' || char(10) ||
        'git config --global http.sslVerify "false"' || char(10) ||
        'git config --system http.sslVerify "false"' || char(10) ||
        '# 信任工作区目录' || char(10) ||
        'git config --global --add safe.directory /workspace' || char(10) ||
        '# 初始化仓库' || char(10) ||
        'git init' || char(10) ||
        'git remote remove origin 2>/dev/null || true' || char(10) ||
        'git remote add origin {{ repository_url }}' || char(10) ||
        '# 拉取代码' || char(10) ||
        'git fetch --depth=1 origin {{ repository_ref }}' || char(10) ||
        'git checkout -B {{ repository_ref }} FETCH_HEAD',
        '克隆代码仓库',
        '[]',
        '[]',
        0,
        '2024-03-16 00:00:00 +0000 UTC',
        '2024-03-16 00:00:00 +0000 UTC'
    ),
    (
        'yrkdm4fc4wlupvd3ne6ea2ywpe',
        '01KNVEJPWVK757139NMNNNCEFE',
        '01KNRDSSJ7RNND7110175N4NR2',
        'golang:1.23 build',
        5,
        '运行 Go 构建',
        'golang:1.25 build',
        'golang:1.23-alpine',
        'set -e' || char(10) ||
        'cd {{ working_dir }}' || char(10) ||
        'mkdir -p dist' || char(10) ||
        'go env -w GOPROXY=https://goproxy.cn,direct' || char(10) ||
        'go build -o dist/',
        '运行 Go 构建',
        '[]',
        '["4smvyi2oq4n2kzrio4zmcqykge"]',
        3,
        '2024-03-16 00:00:00 +0000 UTC',
        '2024-03-16 00:00:00 +0000 UTC'
    )
ON CONFLICT(id) DO UPDATE SET
    pipeline_id = excluded.pipeline_id,
    source_template_stage_id = excluded.source_template_stage_id,
    source_template_stage_name = excluded.source_template_stage_name,
    source_template_stage_version = excluded.source_template_stage_version,
    source_template_stage_description = excluded.source_template_stage_description,
    name = excluded.name,
    image = excluded.image,
    script = excluded.script,
    description = excluded.description,
    artifacts = excluded.artifacts,
    depends_on = excluded.depends_on,
    sort_order = excluded.sort_order,
    created_at = excluded.created_at,
    updated_at = excluded.updated_at;
