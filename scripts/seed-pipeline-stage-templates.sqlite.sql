-- Reusable build-stage templates for the default project.
-- Requires migrations 000029_seed_system and 000030_pipeline_stage_template_library.
INSERT OR IGNORE INTO pipeline_stage (
    id, project_id, kind, name, image, script, description, version, artifacts, created_at, updated_at
) VALUES
    (
        '01KRCWNJVA1DM02TJXZ4STJD01',
        '01KRRKK0K3T519ZQZES3M4QA9Z',
        'template',
        'git clone',
        'alpine/git',
        'set -e' || char(10) ||
        'git config --global http.sslVerify "false"' || char(10) ||
        'git config --system http.sslVerify "false"' || char(10) ||
        'git config --global --add safe.directory /workspace' || char(10) ||
        'git init' || char(10) ||
        'git remote remove origin 2>/dev/null || true' || char(10) ||
        'git remote add origin {{ repository_url }}' || char(10) ||
        'git fetch --depth=1 origin {{ repository_ref }}' || char(10) ||
        'git checkout -B {{ repository_ref }} FETCH_HEAD',
        'Clone source repository',
        1,
        '[{"collector":"command","command":"git rev-parse HEAD","format":"git_object_id","name":"source_commit"}]',
        '2024-03-16T00:00:00Z',
        '2024-03-16T00:00:00Z'
    ),
    (
        '01KRCWNJVA1DM02TJXZ4STJD06',
        '01KRRKK0K3T519ZQZES3M4QA9Z',
        'template',
        'docker build',
        'docker:29.4',
        'set -e' || char(10) ||
        'cd {{ working_dir }}' || char(10) ||
        'docker build -t {{ repository_code }}:{{ runtime_datetime }} -f {{ repository_dockerfile }} .',
        'Build container image',
        1,
        '[{"collector":"docker_image","reference":"{{ repository_code }}:{{ runtime_datetime }}","name":"{{ repository_code }}"}]',
        '2024-03-16T00:00:00Z',
        '2024-03-16T00:00:00Z'
    );

INSERT OR IGNORE INTO pipeline (
    id, project_id, kind, name, description, variable_declarations, version, created_at, updated_at
) VALUES (
    '01KZG83K2MXG08EJ6G48SG38B3',
    '01KRRKK0K3T519ZQZES3M4QA9Z',
    'template',
    '镜像构建流水线',
    '适用于使用 Dockerfile 进制镜像构建的仓库',
    '[]',
    9,
    '2026-08-08T08:33:50.6767901Z',
    '2026-08-08T09:35:00.6543397Z'
);

INSERT OR IGNORE INTO pipeline_stage_reference (
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
        1,
        'Clone source repository',
        'git clone',
        'alpine/git',
        'set -e' || char(10) ||
        'git config --global http.sslVerify "false"' || char(10) ||
        'git config --system http.sslVerify "false"' || char(10) ||
        'git config --global --add safe.directory /workspace' || char(10) ||
        'git init' || char(10) ||
        'git remote remove origin 2>/dev/null || true' || char(10) ||
        'git remote add origin {{ repository_url }}' || char(10) ||
        'git fetch --depth=1 origin {{ repository_ref }}' || char(10) ||
        'git checkout -B {{ repository_ref }} FETCH_HEAD',
        'Clone source repository',
        '[{"collector":"command","command":"git rev-parse HEAD","format":"git_object_id","name":"source_commit"}]',
        '[]',
        1,
        '2026-08-08T09:33:12.7622645Z',
        '2026-08-08T09:33:12.7622645Z'
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
        '2026-08-08T09:33:16.7695648Z',
        '2026-08-08T09:33:16.7695648Z'
    );
