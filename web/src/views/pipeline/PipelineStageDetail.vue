<template>
  <div class="space-y-6">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h1 class="app-detail-page-title break-words">{{ stage?.name || '阶段详情' }}</h1>
      <div class="flex items-center gap-2">
        <button v-if="stage" class="app-button-danger h-9 px-3" @click="openDelete">
          <Trash2 class="size-4" />
          删除
        </button>
        <button class="app-button h-9 px-3" @click="router.push('/pipeline-stage')">
          <ArrowLeft class="size-4" />
          返回
        </button>
      </div>
    </div>

    <AppLoadingState v-if="status === 'loading'" size="section" />
    <p v-else-if="status === 'error'" class="py-16 text-center text-sm text-destructive">
      {{ error || '加载阶段失败' }}
    </p>

    <template v-else-if="stage">
      <section class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">基本信息</h2>
          <button class="app-button-primary h-9 px-3" @click="openBasicEdit">
            <Pencil class="size-4" />
            编辑
          </button>
        </div>
        <dl class="app-detail-info-grid">
          <div class="flex gap-2">
            <dt>名称</dt>
            <dd class="text-foreground">{{ stage.name }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>版本</dt>
            <dd class="text-foreground">v{{ stage.version }}</dd>
          </div>
          <div class="flex gap-2 sm:col-span-2">
            <dt>执行镜像</dt>
            <dd class="min-w-0 break-all text-foreground">{{ stage.image }}</dd>
          </div>
          <div v-if="stage.description" class="flex gap-2 sm:col-span-2">
            <dt>说明</dt>
            <dd class="whitespace-pre-wrap text-foreground">{{ stage.description }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>创建时间</dt>
            <dd class="text-muted-foreground">{{ formatTime(stage.created_at) }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>更新时间</dt>
            <dd class="text-muted-foreground">{{ formatTime(stage.updated_at) }}</dd>
          </div>
        </dl>
      </section>

      <section class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">脚本</h2>
          <div class="flex items-center gap-2">
            <button class="app-button-primary h-9 px-3" @click="openScriptEdit">
              <Pencil class="size-4" />
              编辑
            </button>
            <button v-if="stage.script" class="app-button h-9 px-3" @click="copyScript">
              <Copy class="size-4" />
              复制
            </button>
          </div>
        </div>
        <div v-if="stage.script" class="p-5">
          <MonacoEditor
            :model-value="stage.script"
            language="shell"
            height="200px"
            :readonly="true"
          />
        </div>
        <div v-else class="px-5 py-10 text-center text-sm text-muted-foreground">暂无脚本</div>
      </section>

      <section class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">制品声明</h2>
          <button class="app-button-primary h-9 px-3" @click="openArtifact">
            <Plus class="size-4" />
            添加制品
          </button>
        </div>
        <AppEmptyState v-if="stage.artifacts.length === 0" size="compact" />
        <div v-else class="overflow-x-auto">
          <table class="app-data-table min-w-[720px]">
            <thead>
              <tr>
                <th>名称</th>
                <th>收集器</th>
                <th>引用或命令</th>
                <th class="w-28">操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(artifact, index) in stage.artifacts" :key="artifact.name">
                <td class="text-foreground">{{ artifact.name }}</td>
                <td><AppBadge>{{ artifact.collector }}</AppBadge></td>
                <td class="max-w-xl truncate text-muted-foreground">
                  {{ artifact.collector === 'command' ? artifact.command : artifact.reference }}
                </td>
                <td>
                  <div class="flex items-center gap-3">
                    <button class="app-link" @click="openArtifact(index)">编辑</button>
                    <button class="app-link-danger" @click="removeArtifact(index)">删除</button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </template>

    <AppDialog v-model:open="basicOpen" title="编辑阶段">
      <form class="space-y-4" @submit.prevent="saveBasic">
        <div class="space-y-1.5">
          <label class="app-field-label">名称 <span class="text-destructive">*</span></label>
          <input
            v-model="basicForm.name"
            class="app-input"
            :class="basicErrors.name && 'app-input-error'"
            :aria-invalid="basicErrors.name ? 'true' : undefined"
            @input="basicErrors.name = ''"
          />
          <p v-if="basicErrors.name" class="app-field-error" role="alert">{{ basicErrors.name }}</p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label">执行镜像 <span class="text-destructive">*</span></label>
          <input
            v-model="basicForm.image"
            class="app-input"
            :class="basicErrors.image && 'app-input-error'"
            :aria-invalid="basicErrors.image ? 'true' : undefined"
            @input="basicErrors.image = ''"
          />
          <p v-if="basicErrors.image" class="app-field-error" role="alert">{{ basicErrors.image }}</p>
        </div>
        <div class="space-y-1.5">
          <label class="app-field-label">说明</label>
          <textarea v-model="basicForm.description" rows="3" class="app-textarea" />
        </div>
      </form>
      <template #footer>
        <AppDialogActions :busy="saving" @cancel="basicOpen = false" @confirm="saveBasic" />
      </template>
    </AppDialog>

    <AppDrawer
      :open="scriptOpen"
      title="编辑脚本"
      width-class="w-[min(960px,100vw)]"
      body-class="min-h-0 flex-1 overflow-hidden p-0"
      @update:open="setScriptDrawerOpen"
    >
      <form class="flex h-full min-h-0 flex-col gap-3 p-6" @submit.prevent="saveScript">
        <div class="min-h-0 flex-1">
          <MonacoEditor v-model="scriptForm.script" language="shell" height="100%" />
        </div>
        <p v-if="scriptError" class="app-field-error shrink-0" role="alert">
          {{ scriptError }}
        </p>
      </form>
      <template #footer>
        <AppDialogActions :busy="saving" @cancel="closeScriptDrawer" @confirm="saveScript" />
      </template>
    </AppDrawer>

    <AppDialog v-model:open="artifactOpen" :title="artifactIndex === -1 ? '添加制品' : '编辑制品'">
      <form class="space-y-4" @submit.prevent="saveArtifact">
        <div class="grid gap-4 sm:grid-cols-2">
          <div class="space-y-1.5">
            <label class="app-field-label">名称 <span class="text-destructive">*</span></label>
            <input
              v-model="artifactForm.name"
              class="app-input"
              :class="artifactErrors.name && 'app-input-error'"
              :aria-invalid="artifactErrors.name ? 'true' : undefined"
              @input="artifactErrors.name = ''"
            />
            <p v-if="artifactErrors.name" class="app-field-error" role="alert">{{ artifactErrors.name }}</p>
          </div>
          <div class="space-y-1.5">
            <label class="app-field-label">收集器 <span class="text-destructive">*</span></label>
            <RawValueSelect
              :model-value="artifactForm.collector"
              :values="artifactCollectors"
              :invalid="Boolean(artifactErrors.collector)"
              @update:model-value="changeArtifactCollector"
            />
            <p v-if="artifactErrors.collector" class="app-field-error" role="alert">
              {{ artifactErrors.collector }}
            </p>
          </div>
        </div>
        <div v-if="artifactForm.collector === 'command'" class="grid gap-4 sm:grid-cols-2">
          <div class="space-y-1.5">
            <label class="app-field-label">命令 <span class="text-destructive">*</span></label>
            <input
              v-model="artifactForm.command"
              class="app-input"
              :class="artifactErrors.command && 'app-input-error'"
              :aria-invalid="artifactErrors.command ? 'true' : undefined"
              @input="artifactErrors.command = ''"
            />
            <p v-if="artifactErrors.command" class="app-field-error" role="alert">
              {{ artifactErrors.command }}
            </p>
          </div>
          <div class="space-y-1.5">
            <label class="app-field-label">输出格式 <span class="text-destructive">*</span></label>
            <RawValueSelect
              :model-value="artifactForm.format"
              :values="artifactFormats"
              :invalid="Boolean(artifactErrors.format)"
              @update:model-value="changeArtifactFormat"
            />
            <p v-if="artifactErrors.format" class="app-field-error" role="alert">
              {{ artifactErrors.format }}
            </p>
          </div>
        </div>
        <div v-else class="space-y-1.5">
          <label class="app-field-label">引用 <span class="text-destructive">*</span></label>
          <input
            v-model="artifactForm.reference"
            class="app-input"
            :class="artifactErrors.reference && 'app-input-error'"
            :aria-invalid="artifactErrors.reference ? 'true' : undefined"
            @input="artifactErrors.reference = ''"
          />
          <p v-if="artifactErrors.reference" class="app-field-error" role="alert">
            {{ artifactErrors.reference }}
          </p>
        </div>
      </form>
      <template #footer>
        <AppDialogActions :busy="saving" @cancel="closeArtifact" @confirm="saveArtifact" />
      </template>
    </AppDialog>

    <AppDialog v-model:open="deleteOpen" title="删除阶段">
      <p class="text-sm text-muted-foreground">删除不会影响已引入到流水线的阶段或历史执行记录。</p>
      <p v-if="deleteError" class="app-field-error mt-3" role="alert">{{ deleteError }}</p>
      <template #footer>
        <AppDialogActions
          :busy="saving"
          confirm-label="删除"
          variant="destructive"
          @cancel="deleteOpen = false"
          @confirm="removeStage"
        />
      </template>
    </AppDialog>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft, Copy, Pencil, Plus, Trash2 } from 'lucide-vue-next';
  import { onMounted, reactive, ref, watch } from 'vue';
  import { useRoute, useRouter } from 'vue-router';
  import { pipelineStageApi } from '@/api/pipeline/pipeline_stage';
  import AppBadge from '@/components/AppBadge.vue';
  import AppDialog from '@/components/AppDialog.vue';
  import AppDialogActions from '@/components/AppDialogActions.vue';
  import AppDrawer from '@/components/AppDrawer.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import MonacoEditor from '@/components/MonacoEditor.vue';
  import RawValueSelect, { type RawValue } from '@/components/RawValueSelect.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type {
    ArtifactConfigReq,
    PipelineStageResp,
  } from '@/gen/proto/orbit/v1/pipeline/pipeline_stage';
  import { formatTime } from '@/utils/time';

  type ArtifactError = 'name' | 'collector' | 'reference' | 'command' | 'format';

  const route = useRoute();
  const router = useRouter();
  const toast = useToast();
  const { status, error, execute } = useStatusAsync();
  const { loading: saving, execute: executeSave } = useStatusAsync();
  const stage = ref<PipelineStageResp>();
  const basicOpen = ref(false);
  const scriptOpen = ref(false);
  const artifactOpen = ref(false);
  const deleteOpen = ref(false);
  const artifactIndex = ref(-1);
  const deleteError = ref('');
  const scriptError = ref('');
  const basicForm = reactive({ name: '', image: '', description: '' });
  const basicErrors = reactive({ name: '', image: '' });
  const scriptForm = reactive({ script: '' });
  const artifactForm = reactive<ArtifactConfigReq>({
    name: '',
    collector: 'docker_image',
    reference: '',
    command: '',
    format: '',
  });
  const artifactErrors = reactive<Record<ArtifactError, string>>({
    name: '',
    collector: '',
    reference: '',
    command: '',
    format: '',
  });
  const artifactCollectors = ['docker_image', 'file', 'command'];
  const artifactFormats = ['text', 'git_object_id'];

  const stageId = () => String(route.params.id || '');

  async function fetchStage() {
    await execute(async () => {
      stage.value = await pipelineStageApi.get(stageId());
    });
  }

  function openBasicEdit() {
    if (!stage.value) return;
    Object.assign(basicForm, {
      name: stage.value.name,
      image: stage.value.image,
      description: stage.value.description,
    });
    Object.assign(basicErrors, { name: '', image: '' });
    basicOpen.value = true;
  }

  async function saveBasic() {
    basicErrors.name = basicForm.name.trim() ? '' : '请输入阶段名称';
    basicErrors.image = basicForm.image.trim() ? '' : '请输入执行镜像';
    if (basicErrors.name || basicErrors.image) return;
    try {
      await executeSave(async () => {
        stage.value = await pipelineStageApi.update(stageId(), {
          name: basicForm.name.trim(),
          image: basicForm.image.trim(),
          description: basicForm.description,
        });
        basicOpen.value = false;
        toast.success('阶段已保存');
      });
    } catch (reason) {
      toast.error(reason instanceof Error ? reason.message : '保存阶段失败');
    }
  }

  function openScriptEdit() {
    scriptForm.script = stage.value?.script || '';
    scriptError.value = '';
    scriptOpen.value = true;
  }

  async function copyScript() {
    if (!stage.value) return;
    try {
      await navigator.clipboard.writeText(stage.value.script);
      toast.success('脚本已复制到剪贴板');
    } catch (reason) {
      toast.error(reason instanceof Error ? reason.message : '复制脚本失败');
    }
  }

  function closeScriptDrawer() {
    scriptOpen.value = false;
    scriptError.value = '';
  }

  function setScriptDrawerOpen(open: boolean) {
    if (open) {
      scriptOpen.value = true;
      return;
    }
    closeScriptDrawer();
  }

  async function saveScript() {
    scriptError.value = '';
    try {
      await executeSave(async () => {
        stage.value = await pipelineStageApi.update(stageId(), { script: scriptForm.script.trim() });
        closeScriptDrawer();
        toast.success('脚本已保存');
      });
    } catch (reason) {
      scriptError.value = reason instanceof Error ? reason.message : '保存脚本失败';
    }
  }

  function clearArtifactErrors() {
    for (const key of Object.keys(artifactErrors) as ArtifactError[]) artifactErrors[key] = '';
  }

  function openArtifact(index = -1) {
    artifactIndex.value = index;
    Object.assign(
      artifactForm,
      index === -1
        ? { name: '', collector: 'docker_image', reference: '', command: '', format: '' }
        : { ...stage.value?.artifacts[index] }
    );
    clearArtifactErrors();
    artifactOpen.value = true;
  }

  function closeArtifact() {
    artifactOpen.value = false;
    clearArtifactErrors();
  }

  function changeArtifactCollector(value: RawValue) {
    artifactForm.collector = String(value);
    artifactForm.reference = '';
    artifactForm.command = '';
    artifactForm.format = '';
    clearArtifactErrors();
  }

  function changeArtifactFormat(value: RawValue) {
    artifactForm.format = String(value);
    artifactErrors.format = '';
  }

  function normalizedArtifact(): ArtifactConfigReq {
    return {
      name: artifactForm.name.trim(),
      collector: artifactForm.collector,
      reference: artifactForm.collector === 'command' ? '' : artifactForm.reference.trim(),
      command: artifactForm.collector === 'command' ? artifactForm.command.trim() : '',
      format: artifactForm.collector === 'command' ? artifactForm.format : '',
    };
  }

  async function saveArtifact() {
    clearArtifactErrors();
    if (!artifactForm.name.trim()) artifactErrors.name = '请输入制品名称';
    if (!artifactCollectors.includes(artifactForm.collector)) artifactErrors.collector = '请选择收集器';
    if (artifactForm.collector === 'command' && !artifactForm.command.trim())
      artifactErrors.command = '请输入制品命令';
    if (artifactForm.collector === 'command' && !artifactForm.format)
      artifactErrors.format = '请选择输出格式';
    if (artifactForm.collector !== 'command' && !artifactForm.reference.trim())
      artifactErrors.reference = '请输入制品引用';
    if (Object.values(artifactErrors).some(Boolean) || !stage.value) return;

    const artifact = normalizedArtifact();
    const artifacts =
      artifactIndex.value === -1
        ? [...stage.value.artifacts, artifact]
        : stage.value.artifacts.map((item, index) => (index === artifactIndex.value ? artifact : item));
    try {
      await executeSave(async () => {
        stage.value = await pipelineStageApi.update(stageId(), { artifacts: { items: artifacts } });
        closeArtifact();
        toast.success('制品声明已保存');
      });
    } catch (reason) {
      toast.error(reason instanceof Error ? reason.message : '保存制品声明失败');
    }
  }

  async function removeArtifact(index: number) {
    if (!stage.value) return;
    try {
      await executeSave(async () => {
        stage.value = await pipelineStageApi.update(stageId(), {
          artifacts: { items: stage.value?.artifacts.filter((_, itemIndex) => itemIndex !== index) || [] },
        });
        toast.success('制品声明已删除');
      });
    } catch (reason) {
      toast.error(reason instanceof Error ? reason.message : '删除制品声明失败');
    }
  }

  function openDelete() {
    deleteError.value = '';
    deleteOpen.value = true;
  }

  async function removeStage() {
    try {
      await executeSave(async () => {
        await pipelineStageApi.delete(stageId());
        toast.success('阶段已删除');
        await router.replace('/pipeline-stage');
      });
    } catch (reason) {
      deleteError.value = reason instanceof Error ? reason.message : '删除阶段失败';
    }
  }

  watch(() => route.params.id, () => void fetchStage());
  onMounted(() => void fetchStage());
</script>
