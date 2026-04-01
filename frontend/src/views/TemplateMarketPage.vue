<template>
  <div class="template-market-page">
    <a-page-header title="模板市场" sub-title="选择内置模板快速创建 CI 项目">
      <template #extra>
        <a-button type="primary" @click="showCreateModal = true">
          <template #icon><PlusOutlined /></template>
          自定义模板
        </a-button>
      </template>
    </a-page-header>

    <a-spin :spinning="loading">
      <div class="template-grid">
        <a-card
          v-for="template in templates"
          :key="template.id"
          hoverable
          class="template-card"
          @click="handleTemplateClick(template)"
        >
          <template #title>
            <div class="template-title">
              <span>{{ template.name }}</span>
              <a-tag v-if="template.is_builtin" color="blue">内置</a-tag>
            </div>
          </template>

          <div class="template-content">
            <p class="template-description">{{ template.description }}</p>
            
            <div class="template-meta">
              <a-space>
                <span>
                  <CodeOutlined />
                  {{ template.variable_declarations.length }} 个变量
                </span>
                <span>
                  <ClockCircleOutlined />
                  {{ formatDate(template.created_at) }}
                </span>
              </a-space>
            </div>

            <div class="template-actions">
              <a-button type="primary" size="small" @click.stop="handleCreateProject(template)">
                <template #icon><RocketOutlined /></template>
                创建项目
              </a-button>
              <a-button size="small" @click.stop="handleViewDetail(template)">
                查看详情
              </a-button>
            </div>
          </div>
        </a-card>
      </div>

      <a-empty v-if="!loading && templates.length === 0" description="暂无模板" />
    </a-spin>

    <!-- 创建项目弹窗 -->
    <a-modal
      v-model:open="showProjectModal"
      title="基于模板创建项目"
      width="600px"
      @ok="handleProjectSubmit"
      @cancel="projectForm = {}"
    >
      <a-form :model="projectForm" :label-col="{ span: 6 }" :wrapper-col="{ span: 18 }">
        <a-form-item label="项目名称" required>
          <a-input v-model:value="projectForm.name" placeholder="输入项目名称" />
        </a-form-item>

        <a-form-item label="仓库地址" required>
          <a-input v-model:value="projectForm.repository_url" placeholder="https://github.com/user/repo.git" />
        </a-form-item>

        <a-form-item label="Git 凭据" required>
          <a-select v-model:value="projectForm.git_credential_id" placeholder="选择 Git 凭据">
            <a-select-option v-for="cred in credentials" :key="cred.id" :value="cred.id">
              {{ cred.name }} ({{ credentialTypeLabels[cred.type] }})
            </a-select-option>
          </a-select>
          <div style="margin-top: 8px">
            <a-button type="link" size="small" @click="goToCredentials">
              <template #icon><PlusOutlined /></template>
              创建新凭据
            </a-button>
          </div>
        </a-form-item>

        <a-form-item label="分支过滤">
          <a-input v-model:value="projectForm.branch_filter" placeholder="main,develop (留空表示所有分支)" />
        </a-form-item>

        <a-divider>模板变量配置</a-divider>

        <a-form-item
          v-for="varDecl in selectedTemplate?.variable_declarations || []"
          :key="varDecl.name"
          :label="varDecl.name"
          :required="varDecl.required"
        >
          <a-input
            v-model:value="projectForm.variables[varDecl.name]"
            :placeholder="varDecl.default || varDecl.description"
          />
          <div v-if="varDecl.description" class="variable-hint">
            {{ varDecl.description }}
            <span v-if="!varDecl.required && varDecl.default"> (默认: {{ varDecl.default }})</span>
          </div>
        </a-form-item>
      </a-form>
    </a-modal>

    <!-- 自定义模板弹窗 -->
    <a-modal
      v-model:open="showCreateModal"
      title="创建自定义模板"
      width="800px"
      @ok="handleCreateTemplate"
      @cancel="customTemplateForm = {}"
    >
      <a-form :model="customTemplateForm" :label-col="{ span: 4 }" :wrapper-col="{ span: 20 }">
        <a-form-item label="模板名称" required>
          <a-input v-model:value="customTemplateForm.name" placeholder="输入模板名称" />
        </a-form-item>

        <a-form-item label="描述">
          <a-textarea v-model:value="customTemplateForm.description" :rows="2" placeholder="模板描述" />
        </a-form-item>

        <a-form-item label="Pipeline YAML" required>
          <a-textarea
            v-model:value="customTemplateForm.content"
            :rows="12"
            placeholder="version: v1&#10;steps:&#10;  - name: build&#10;    image: alpine&#10;    commands:&#10;      - echo 'Building'"
            style="font-family: monospace"
          />
        </a-form-item>
      </a-form>
    </a-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { message } from 'ant-design-vue'
import {
  PlusOutlined,
  CodeOutlined,
  ClockCircleOutlined,
  RocketOutlined,
} from '@ant-design/icons-vue'
import { pipelineTemplateApi, projectApi, ciCredentialApi } from '@/api'
import { credentialTypeLabels } from '@/types/api'
import type { PipelineTemplate, Credential } from '@/types/api'

const router = useRouter()

const loading = ref(false)
const templates = ref<PipelineTemplate[]>([])
const credentials = ref<Credential[]>([])

const showProjectModal = ref(false)
const showCreateModal = ref(false)
const selectedTemplate = ref<PipelineTemplate | null>(null)

interface ProjectFormData {
  name: string
  repository_url: string
  pipeline_template_id: string
  git_credential_id: string
  branch_filter: string
  variables: Record<string, string>
}

interface CustomTemplateFormData {
  name: string
  description: string
  content: string
}

const projectForm = ref<ProjectFormData>({
  name: '',
  repository_url: '',
  pipeline_template_id: '',
  git_credential_id: '',
  branch_filter: '',
  variables: {},
})

const customTemplateForm = ref<CustomTemplateFormData>({
  name: '',
  description: '',
  content: '',
})

const loadTemplates = async () => {
  loading.value = true
  try {
    templates.value = await pipelineTemplateApi.list()
  } catch (error) {
    message.error(error instanceof Error ? error.message : '加载模板失败')
  } finally {
    loading.value = false
  }
}

const loadCredentials = async () => {
  try {
    credentials.value = await ciCredentialApi.list()
  } catch (error) {
    message.error(error instanceof Error ? error.message : '加载凭据失败')
  }
}

const handleTemplateClick = (template: PipelineTemplate) => {
  router.push(`/ci/templates/${template.id}`)
}

const handleViewDetail = (template: PipelineTemplate) => {
  router.push(`/ci/templates/${template.id}`)
}

const handleCreateProject = (template: PipelineTemplate) => {
  selectedTemplate.value = template
  projectForm.value = {
    name: '',
    repository_url: '',
    pipeline_template_id: template.id,
    git_credential_id: '',
    branch_filter: '',
    variables: {},
  }
  
  // 初始化变量默认值
  template.variable_declarations.forEach((varDecl) => {
    if (varDecl.default) {
      projectForm.value.variables[varDecl.name] = varDecl.default
    }
  })
  
  showProjectModal.value = true
  loadCredentials()
}

const handleProjectSubmit = async () => {
  if (!projectForm.value.name || !projectForm.value.repository_url || !projectForm.value.git_credential_id) {
    message.warning('请填写必填项')
    return
  }

  try {
    const project = await projectApi.create({
      name: projectForm.value.name,
      repository_url: projectForm.value.repository_url,
      pipeline_template_id: projectForm.value.pipeline_template_id,
      git_credential_id: projectForm.value.git_credential_id,
      branch_filter: projectForm.value.branch_filter || undefined,
      variable_overrides: projectForm.value.variables,
    })
    message.success('项目创建成功')
    showProjectModal.value = false
    router.push(`/ci/projects/${project.id}`)
  } catch (error) {
    message.error(error instanceof Error ? error.message : '创建项目失败')
  }
}

const handleCreateTemplate = async () => {
  if (!customTemplateForm.value.name || !customTemplateForm.value.content) {
    message.warning('请填写必填项')
    return
  }

  try {
    const template = await pipelineTemplateApi.create({
      name: customTemplateForm.value.name,
      description: customTemplateForm.value.description || '',
      content: customTemplateForm.value.content,
      variable_declarations: [],
    })
    message.success('模板创建成功')
    showCreateModal.value = false
    await loadTemplates()
    router.push(`/ci/templates/${template.id}`)
  } catch (error) {
    message.error(error instanceof Error ? error.message : '创建模板失败')
  }
}

const goToCredentials = () => {
  showProjectModal.value = false
  router.push('/ci/credentials')
}

const formatDate = (dateStr: string) => {
  return new Date(dateStr).toLocaleDateString('zh-CN')
}

onMounted(() => {
  loadTemplates()
})
</script>

<style scoped>
.template-market-page {
  padding: 24px;
}

.template-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
  gap: 24px;
  margin-top: 24px;
}

.template-card {
  transition: all 0.3s;
}

.template-card:hover {
  transform: translateY(-4px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

.template-title {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.template-content {
  min-height: 180px;
  display: flex;
  flex-direction: column;
}

.template-description {
  color: rgba(0, 0, 0, 0.65);
  margin-bottom: 16px;
  flex: 1;
  min-height: 60px;
}

.template-meta {
  color: rgba(0, 0, 0, 0.45);
  font-size: 13px;
  margin-bottom: 16px;
  padding-top: 12px;
  border-top: 1px solid #f0f0f0;
}

.template-actions {
  display: flex;
  gap: 8px;
}

.variable-hint {
  font-size: 12px;
  color: rgba(0, 0, 0, 0.45);
  margin-top: 4px;
}
</style>
