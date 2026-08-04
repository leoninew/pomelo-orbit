-- Seed (non-domain): pipeline demo data (no application/route demo)
-- Ref: docs/analyze/20260724-domain-split-consensus-共识.md

INSERT OR IGNORE INTO repository (
    id, name, code, repository_type, repository_url, git_credential_id, variable_overrides, default_branch, project_id, created_at, updated_at
) VALUES (
    '01KNNRBH52BQJYT9487B2H8N62',
    'golang/example',
    'golang-example',
    'remote_git',
    'https://github.com/golang/example',
    NULL,
    '[]',
    'master',
    '01KRRKK0K3T519ZQZES3M4QA9Z',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT OR IGNORE INTO repository (
    id, name, code, repository_type, repository_url, git_credential_id, variable_overrides, default_branch, project_id, created_at, updated_at
) VALUES (
    '01KP0JZFQQA2Z77FRRVH35BYF6',
    'docker/awesome-compose',
    'awesome-compose',
    'remote_git',
    'https://github.com/docker/awesome-compose',
    NULL,
    '[]',
    'master',
    '01KRRKK0K3T519ZQZES3M4QA9Z',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT OR IGNORE INTO pipeline_stage (
    id, name, image, script, artifacts, description, version, project_id, created_at, updated_at
) VALUES (
    '01KZ5A17696GZ6NS5BS6VJGR9B',
    'git clone',
    'alpine/git',
    'set -e
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
git checkout -B {{ repository_ref }} FETCH_HEAD',
    NULL,
    '克隆代码仓库',
    8,
    '01KRRKK0K3T519ZQZES3M4QA9Z',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT OR IGNORE INTO pipeline_stage (
    id, name, image, script, artifacts, description, version, project_id, created_at, updated_at
) VALUES (
    '01KNRANZDR4PASATAXKTBBTRX9',
    'golang:1.23 test',
    'golang:1.23-alpine',
    'set -e
cd {{ working_dir }}
go env -w GOPROXY=https://goproxy.cn,direct
go test ./...',
    NULL,
    '运行 Go 单元测试',
    3,
    '01KRRKK0K3T519ZQZES3M4QA9Z',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT OR IGNORE INTO pipeline_stage (
    id, name, image, script, artifacts, description, version, project_id, created_at, updated_at
) VALUES (
    '01KNRDSSJ7RNND7110175N4NR2',
    'golang:1.23 build',
    'golang:1.23-alpine',
    'set -e
cd {{ working_dir }}
mkdir -p dist
go env -w GOPROXY=https://goproxy.cn,direct
go build -o dist/',
    NULL,
    '运行 Go 构建',
    5,
    '01KRRKK0K3T519ZQZES3M4QA9Z',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT OR IGNORE INTO pipeline_stage (
    id, name, image, script, artifacts, description, version, project_id, created_at, updated_at
) VALUES (
    '01KNRKNAHG3EBS07VBK2YY5ZQN',
    'golang:1.23 lint',
    'golang:1.23-alpine',
    'set -e
cd {{ working_dir }}
go env -w GOPROXY=https://goproxy.cn,direct
go install golang.org/x/lint/golint@latest
golint ./...',
    NULL,
    '运行 Go 代码质量',
    3,
    '01KRRKK0K3T519ZQZES3M4QA9Z',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT OR IGNORE INTO pipeline_stage (
    id, name, image, script, artifacts, description, version, project_id, created_at, updated_at
) VALUES (
    '01KP0K4ZTV60PM2XDT11MTX2MY',
    'docker build',
    'docker:29.4',
    'set -e
cd {{ working_dir }}
docker build -t {{ repository_code }}:{{ runtime_datetime }} -f {{ repository_dockerfile }} .
# docker push {{ repository_code }}:{{ runtime_datetime }}',
    '[{"type": "docker_image", "path": "{{ repository_code }}:{{ runtime_datetime }}", "name": "{{ repository_code }}"}]',
    '通用镜像构建+推送',
    16,
    '01KRRKK0K3T519ZQZES3M4QA9Z',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT OR IGNORE INTO pipeline_template (
    id, name, description, variable_declarations, version, project_id, created_at, updated_at
) VALUES (
    '01KNVEJPWVK757139NMNNNCEFE',
    'Go 构建流水线',
    '- test & lint
- build',
    '[{"name":"working_dir","description":"工作目录","default":".","value":null,"secret":false,"source":"template_custom","editable":true}]',
    7,
    '01KRRKK0K3T519ZQZES3M4QA9Z',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT OR IGNORE INTO pipeline_template (
    id, name, description, variable_declarations, version, project_id, created_at, updated_at
) VALUES (
    '01KP0K6W1YW73REFQE8YAVTPMN',
    '通用容器镜像流水线',
    '- docker build',
    '[{"name":"working_dir","description":"","default":".","value":null,"secret":false,"source":"template_custom","editable":true},{"name":"repository_dockerfile","description":"","default":"Dockerfile","value":null,"secret":false,"source":"template_custom","editable":true}]',
    24,
    '01KRRKK0K3T519ZQZES3M4QA9Z',
    '2024-03-16T00:00:00Z',
    '2024-03-16T00:00:00Z'
);

INSERT OR IGNORE INTO pipeline_template_stage (
    id, template_id, stage_id, stage_name, stage_version, depends_on, sort_order
) VALUES (
    '01KRCWNJVA1DM02TJXZ4STJCCK',
    '01KNVEJPWVK757139NMNNNCEFE',
    '01KZ5A17696GZ6NS5BS6VJGR9B',
    'git clone',
    8,
    '[]',
    0
);

INSERT OR IGNORE INTO pipeline_template_stage (
    id, template_id, stage_id, stage_name, stage_version, depends_on, sort_order
) VALUES (
    '01KRCWNJVA1DM02TJXZ4STJCCM',
    '01KNVEJPWVK757139NMNNNCEFE',
    '01KNRANZDR4PASATAXKTBBTRX9',
    'golang:1.23 test',
    3,
    '["01KZ5A17696GZ6NS5BS6VJGR9B"]',
    1
);

INSERT OR IGNORE INTO pipeline_template_stage (
    id, template_id, stage_id, stage_name, stage_version, depends_on, sort_order
) VALUES (
    '01KRCWNJVA1DM02TJXZ4STJCCN',
    '01KNVEJPWVK757139NMNNNCEFE',
    '01KNRKNAHG3EBS07VBK2YY5ZQN',
    'golang:1.23 lint',
    3,
    '["01KZ5A17696GZ6NS5BS6VJGR9B"]',
    2
);

INSERT OR IGNORE INTO pipeline_template_stage (
    id, template_id, stage_id, stage_name, stage_version, depends_on, sort_order
) VALUES (
    '01KRCWNJVA1DM02TJXZ4STJCCP',
    '01KNVEJPWVK757139NMNNNCEFE',
    '01KNRDSSJ7RNND7110175N4NR2',
    'golang:1.23 build',
    5,
    '["01KNRANZDR4PASATAXKTBBTRX9"]',
    3
);

INSERT OR IGNORE INTO pipeline_template_stage (
    id, template_id, stage_id, stage_name, stage_version, depends_on, sort_order
) VALUES (
    '01KRCYJJF8GNWTA76CE2M4QXAZ',
    '01KP0K6W1YW73REFQE8YAVTPMN',
    '01KZ5A17696GZ6NS5BS6VJGR9B',
    'git clone',
    8,
    '[]',
    0
);

INSERT OR IGNORE INTO pipeline_template_stage (
    id, template_id, stage_id, stage_name, stage_version, depends_on, sort_order
) VALUES (
    '01KRCYJJF8GNWTA76CE2M4QXB0',
    '01KP0K6W1YW73REFQE8YAVTPMN',
    '01KP0K4ZTV60PM2XDT11MTX2MY',
    'docker build',
    16,
    '["01KZ5A17696GZ6NS5BS6VJGR9B"]',
    1
);
