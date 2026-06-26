<template>
  <div class="space-y-6">
    <StepperRoot v-model="currentStep" class="space-y-6" :linear="true">
      <div class="grid gap-3 md:grid-cols-3">
        <StepperItem
          v-for="step in steps"
          :key="step.value"
          v-slot="{ state }"
          :step="step.value"
          :completed="currentStep > step.value"
          :disabled="step.value > maxStep"
        >
          <StepperTrigger
            class="flex w-full items-start gap-3 rounded-xl border border-border bg-background p-3 text-left transition-colors disabled:cursor-not-allowed disabled:opacity-60"
            :class="state === 'active' ? 'border-primary bg-primary/5' : ''"
          >
            <StepperIndicator
              class="flex size-7 shrink-0 items-center justify-center rounded-full border border-border text-sm font-semibold"
              :class="state === 'active' || state === 'completed' ? 'border-primary text-primary' : ''"
            >
              {{ step.value }}
            </StepperIndicator>
            <div class="min-w-0 space-y-1">
              <StepperTitle class="text-sm font-semibold text-foreground">
                {{ step.title }}
              </StepperTitle>
              <StepperDescription class="text-xs text-muted-foreground">
                {{ step.description }}
              </StepperDescription>
            </div>
          </StepperTrigger>
          <StepperSeparator class="hidden" />
        </StepperItem>
      </div>

      <div v-if="currentStep === 1" class="space-y-4">
        <div class="grid gap-3 md:grid-cols-2">
          <button
            v-for="method in createMethods"
            :key="method.value"
            class="rounded-xl border border-border bg-background p-4 text-left transition-colors hover:border-primary/60"
            :class="selectedMethod === method.value ? 'border-primary bg-primary/5' : ''"
            @click="selectedMethod = method.value"
          >
            <div class="flex items-start justify-between gap-3">
              <div class="space-y-1">
                <h3 class="font-semibold text-foreground">{{ method.title }}</h3>
                <p class="text-sm text-muted-foreground">{{ method.description }}</p>
              </div>
              <span
                v-if="method.badge"
                class="rounded-full bg-muted px-2 py-1 text-xs text-muted-foreground"
              >
                {{ method.badge }}
              </span>
            </div>
          </button>
        </div>

        <div v-if="selectedMethod === 'compose' || selectedMethod === 'import'" class="app-tip">
          {{ t('application.createWizard.unsupportedHint') }}
        </div>
      </div>

      <form v-else-if="currentStep === 2" ref="currentFormRef" class="space-y-5" @submit.prevent>
        <ApplicationFormFields
          v-if="selectedMethod === 'blank'"
          :form="blankForm"
          :errors="blankErrors"
          @update:form="Object.assign(blankForm, $event)"
        />

        <template v-else>
          <div class="grid gap-4 md:grid-cols-2">
            <div class="space-y-1.5">
              <label class="app-field-label block" for="image-create-name">
                {{ t('application.name') }}
                <span class="text-destructive">*</span>
              </label>
              <input
                id="image-create-name"
                v-model="imageForm.name"
                required
                type="text"
                class="app-input"
                :placeholder="t('application.namePlaceholder')"
              />
            </div>

            <div class="space-y-1.5">
              <label class="app-field-label block" for="image-create-code">
                {{ t('application.code') }}
                <span class="text-destructive">*</span>
              </label>
              <input
                id="image-create-code"
                v-model="imageForm.code"
                required
                pattern="^[a-z][a-z0-9-]*$"
                type="text"
                class="app-input"
                :placeholder="t('application.codePlaceholder')"
              />
              <p class="app-field-hint">{{ t('application.codeHint') }}</p>
            </div>

            <div class="space-y-1.5">
              <label class="app-field-label block" for="image-create-image">
                {{ t('application.createWizard.image') }}
                <span class="text-destructive">*</span>
              </label>
              <input
                id="image-create-image"
                v-model="imageForm.image"
                required
                type="text"
                class="app-input"
                :placeholder="t('application.createWizard.imagePlaceholder')"
              />
            </div>

            <div class="space-y-1.5">
              <label class="app-field-label block" for="image-create-port">
                {{ t('application.detail.fields.containerPort') }}
                <span class="text-destructive">*</span>
              </label>
              <input
                id="image-create-port"
                v-model="imageForm.containerPort"
                required
                min="1"
                max="65535"
                type="number"
                class="app-input"
                :placeholder="t('application.detail.placeholders.port')"
              />
            </div>

            <div class="space-y-1.5 md:col-span-2">
              <label class="app-field-label block">{{ t('application.imagePullPolicy') }}</label>
              <SelectControl v-model="imageForm.imagePullPolicy" :options="imagePullPolicyOptions" />
            </div>
          </div>

          <div class="space-y-3 rounded-xl border border-border p-4">
            <div class="flex items-center justify-between gap-3">
              <div>
                <h3 class="font-semibold text-foreground">
                  {{ t('application.createWizard.envVars') }}
                </h3>
                <p class="text-sm text-muted-foreground">
                  {{ t('application.createWizard.envVarsHint') }}
                </p>
              </div>
              <button class="app-button h-9 px-3" type="button" @click="addEnvRow">
                {{ t('common.add') }}
              </button>
            </div>
            <div class="space-y-2">
              <div
                v-for="(row, index) in imageForm.envVars"
                :key="index"
                class="grid gap-2 md:grid-cols-[1fr_1fr_auto]"
              >
                <input
                  v-model="row.key"
                  type="text"
                  class="app-input"
                  :placeholder="t('application.createWizard.envKeyPlaceholder')"
                />
                <input
                  v-model="row.value"
                  type="text"
                  class="app-input"
                  :placeholder="t('application.createWizard.envValuePlaceholder')"
                />
                <button class="app-button h-10 px-3" type="button" @click="removeEnvRow(index)">
                  {{ t('common.remove') }}
                </button>
              </div>
            </div>
            <p v-if="formErrors.envVars" class="app-field-error">{{ formErrors.envVars }}</p>
          </div>

          <div class="space-y-3 rounded-xl border border-border p-4">
            <div class="flex items-center justify-between gap-3">
              <div>
                <h3 class="font-semibold text-foreground">
                  {{ t('application.createWizard.volumes') }}
                </h3>
                <p class="text-sm text-muted-foreground">
                  {{ t('application.createWizard.volumesHint') }}
                </p>
              </div>
              <button class="app-button h-9 px-3" type="button" @click="addVolumeRow">
                {{ t('common.add') }}
              </button>
            </div>
            <div class="space-y-2">
              <div
                v-for="(row, index) in imageForm.volumes"
                :key="index"
                class="grid gap-2 md:grid-cols-[1fr_1fr_auto]"
              >
                <input
                  v-model="row.hostPath"
                  type="text"
                  class="app-input"
                  :placeholder="t('application.createWizard.hostPathPlaceholder')"
                />
                <input
                  v-model="row.containerPath"
                  type="text"
                  class="app-input"
                  :placeholder="t('application.createWizard.containerPathPlaceholder')"
                />
                <button class="app-button h-10 px-3" type="button" @click="removeVolumeRow(index)">
                  {{ t('common.remove') }}
                </button>
              </div>
            </div>
            <p v-if="formErrors.volumes" class="app-field-error">{{ formErrors.volumes }}</p>
          </div>

          <div class="space-y-1.5">
            <label class="app-field-label block" for="image-create-domain">
              {{ t('application.createWizard.routeDomain') }}
            </label>
            <input
              id="image-create-domain"
              v-model="imageForm.routeDomain"
              type="text"
              class="app-input"
              :placeholder="t('application.createWizard.routeDomainPlaceholder')"
            />
            <p class="app-field-hint">{{ t('application.createWizard.routeDomainHint') }}</p>
          </div>
        </template>
      </form>

      <div v-else class="space-y-4">
        <div v-if="selectedMethod === 'blank'" class="app-tip">
          {{ t('application.createWizard.blankConfirmHint') }}
        </div>

        <template v-else>
          <div class="grid gap-3 md:grid-cols-2">
            <div class="app-tip">
              <div class="font-semibold text-foreground">{{ t('application.name') }}</div>
              <div>{{ imageForm.name }}</div>
            </div>
            <div class="app-tip">
              <div class="font-semibold text-foreground">{{ t('application.code') }}</div>
              <div>{{ imageForm.code }}</div>
            </div>
            <div class="app-tip">
              <div class="font-semibold text-foreground">{{ t('application.createWizard.image') }}</div>
              <div>{{ imageForm.image }}</div>
            </div>
            <div class="app-tip">
              <div class="font-semibold text-foreground">
                {{ t('application.detail.fields.containerPort') }}
              </div>
              <div>{{ imageForm.containerPort }}</div>
            </div>
          </div>

          <div class="rounded-xl border border-border bg-background p-4">
            <div class="mb-2 text-sm font-semibold text-foreground">docker-compose.yml</div>
            <pre class="max-h-72 overflow-auto rounded-xl bg-muted/40 p-3 text-xs text-foreground">{{ composePreview }}</pre>
          </div>

          <div v-if="envPreview" class="rounded-xl border border-border bg-background p-4">
            <div class="mb-2 text-sm font-semibold text-foreground">.env</div>
            <pre class="max-h-48 overflow-auto rounded-xl bg-muted/40 p-3 text-xs text-foreground">{{ envPreview }}</pre>
          </div>
        </template>
      </div>
    </StepperRoot>

    <div v-if="submitError" class="rounded-xl border border-destructive/30 bg-destructive/5 p-4 text-sm">
      <p class="text-destructive">{{ submitError }}</p>
      <button v-if="createdApplication" class="app-button mt-3" @click="goCreatedApplicationDetail">
        {{ t('application.createWizard.goDetailFix') }}
      </button>
    </div>

    <div class="flex justify-end gap-2 border-t border-border pt-4">
      <button class="app-button" :disabled="submitting" @click="router.push('/cd/applications')">
        {{ t('common.cancel') }}
      </button>
      <button v-if="currentStep > 1" class="app-button" :disabled="submitting" @click="currentStep -= 1">
        {{ t('application.createWizard.previous') }}
      </button>
      <button
        v-if="currentStep < 3"
        class="app-button-primary"
        :disabled="submitting || selectedMethod === 'compose' || selectedMethod === 'import'"
        @click="handleNext"
      >
        {{ t('application.createWizard.next') }}
      </button>
      <button v-else class="app-button-primary" :disabled="submitting" type="button" @click="handleSubmit">
        {{ submitButtonText }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, reactive, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRouter } from 'vue-router';
  import { applicationApi } from '@/api/cd/application';
  import ApplicationFormFields from '@/components/ApplicationFormFields.vue';
  import SelectControl from '@/components/SelectControl.vue';
  import type { Application, ApplicationFormState } from '@/types/cd/application';
  import {
    buildEnvFile,
    buildImageCompose,
    imageCreateServiceName,
    isValidEnvKey,
    normalizeEnvRows,
    normalizeVolumeRows,
    type ApplicationEnvRow,
    type ApplicationVolumeRow,
  } from '@/utils/applicationCompose';
  import {
    StepperDescription,
    StepperIndicator,
    StepperItem,
    StepperRoot,
    StepperSeparator,
    StepperTitle,
    StepperTrigger,
  } from 'reka-ui';

  type CreateMethod = 'image' | 'compose' | 'import' | 'blank';

  const props = defineProps<{
    projectId?: string | null;
  }>();

  const emit = defineEmits<{
    created: [];
  }>();

  const { t } = useI18n();
  const router = useRouter();

  const currentStep = ref(1);
  const selectedMethod = ref<CreateMethod>('image');
  const currentFormRef = ref<HTMLFormElement>();
  const submitting = ref(false);
  const submitError = ref('');
  const createdApplication = ref<Application | null>(null);
  const blankForm = reactive<ApplicationFormState>({
    name: '',
    code: '',
    image_pull_policy: 'missing',
    route_managed: false,
  });
  const blankErrors = reactive({ name: '', code: '' });
  const imageForm = reactive({
    name: '',
    code: '',
    image: '',
    containerPort: '80',
    imagePullPolicy: 'missing',
    envVars: [{ key: '', value: '' }] as ApplicationEnvRow[],
    volumes: [{ hostPath: '', containerPath: '' }] as ApplicationVolumeRow[],
    routeDomain: '',
  });
  const formErrors = reactive({ envVars: '', volumes: '' });

  const maxStep = computed(() => {
    if (selectedMethod.value === 'compose' || selectedMethod.value === 'import') {
      return 1;
    }
    return 3;
  });
  const normalizedEnvVars = computed(() => normalizeEnvRows(imageForm.envVars));
  const normalizedVolumes = computed(() => normalizeVolumeRows(imageForm.volumes));
  const containerPort = computed(() => Number(imageForm.containerPort));
  const composePreview = computed(() =>
    buildImageCompose({
      image: imageForm.image,
      containerPort: Number.isFinite(containerPort.value) ? containerPort.value : 80,
      envVars: imageForm.envVars,
      volumes: imageForm.volumes,
    })
  );
  const envPreview = computed(() => buildEnvFile(imageForm.envVars));
  const submitButtonText = computed(() => {
    if (submitting.value) {
      return t('application.createWizard.creating');
    }
    return t('application.createWizard.confirmCreate');
  });

  const steps = computed(() => [
    {
      value: 1,
      title: t('application.createWizard.steps.method'),
      description: t('application.createWizard.steps.methodDesc'),
    },
    {
      value: 2,
      title: t('application.createWizard.steps.config'),
      description: t('application.createWizard.steps.configDesc'),
    },
    {
      value: 3,
      title: t('application.createWizard.steps.confirm'),
      description: t('application.createWizard.steps.confirmDesc'),
    },
  ]);

  const createMethods = computed(() => [
    {
      value: 'image' as CreateMethod,
      title: t('application.createWizard.methods.image'),
      description: t('application.createWizard.methods.imageDesc'),
      badge: t('application.createWizard.availableNow'),
    },
    {
      value: 'compose' as CreateMethod,
      title: t('application.createWizard.methods.compose'),
      description: t('application.createWizard.methods.composeDesc'),
      badge: t('application.createWizard.later'),
    },
    {
      value: 'import' as CreateMethod,
      title: t('application.createWizard.methods.import'),
      description: t('application.createWizard.methods.importDesc'),
      badge: t('application.createWizard.useCurrentImport'),
    },
    {
      value: 'blank' as CreateMethod,
      title: t('application.createWizard.methods.blank'),
      description: t('application.createWizard.methods.blankDesc'),
      badge: t('application.createWizard.availableNow'),
    },
  ]);

  const imagePullPolicyOptions = computed(() => [
    { value: 'missing', label: t('application.imagePullPolicyOptions.missing') },
    { value: 'always', label: t('application.imagePullPolicyOptions.always') },
    { value: 'never', label: t('application.imagePullPolicyOptions.never') },
  ]);

  function addEnvRow() {
    imageForm.envVars.push({ key: '', value: '' });
  }

  function removeEnvRow(index: number) {
    imageForm.envVars.splice(index, 1);
    if (imageForm.envVars.length === 0) {
      addEnvRow();
    }
  }

  function addVolumeRow() {
    imageForm.volumes.push({ hostPath: '', containerPath: '' });
  }

  function removeVolumeRow(index: number) {
    imageForm.volumes.splice(index, 1);
    if (imageForm.volumes.length === 0) {
      addVolumeRow();
    }
  }

  function handleNext() {
    submitError.value = '';
    if (currentStep.value === 1) {
      currentStep.value = 2;
      return;
    }
    if (selectedMethod.value === 'blank' && validateBlankStep()) {
      currentStep.value = 3;
      return;
    }
    if (selectedMethod.value === 'image' && validateImageStep()) {
      currentStep.value = 3;
    }
  }

  function validateBlankStep() {
    blankErrors.name = blankForm.name.trim() ? '' : t('application.validation.nameRequired');
    blankErrors.code = /^[a-z][a-z0-9-]*$/.test(blankForm.code)
      ? ''
      : t('application.validation.codeInvalid');
    return !blankErrors.name && !blankErrors.code;
  }

  function validateImageStep() {
    formErrors.envVars = '';
    formErrors.volumes = '';
    if (currentFormRef.value && !currentFormRef.value.reportValidity()) {
      return false;
    }
    if (!imageForm.name.trim()) {
      return false;
    }
    if (!/^[a-z][a-z0-9-]*$/.test(imageForm.code)) {
      return false;
    }
    if (!imageForm.image.trim()) {
      return false;
    }
    if (!Number.isInteger(containerPort.value) || containerPort.value < 1 || containerPort.value > 65535) {
      return false;
    }
    for (const row of normalizedEnvVars.value) {
      if (!isValidEnvKey(row.key)) {
        formErrors.envVars = t('application.validation.envKeyInvalid');
        return false;
      }
      if (row.value.includes('\n')) {
        formErrors.envVars = t('application.validation.envValueInvalid');
        return false;
      }
    }
    for (const row of normalizedVolumes.value) {
      if (!row.hostPath || !row.containerPath) {
        formErrors.volumes = t('application.validation.volumeInvalid');
        return false;
      }
      if (!row.containerPath.startsWith('/')) {
        formErrors.volumes = t('application.validation.containerPathInvalid');
        return false;
      }
    }
    return true;
  }

  async function handleSubmit() {
    if (selectedMethod.value === 'blank') {
      await submitBlankApplication();
      return;
    }
    await submitImageApplication();
  }

  async function submitBlankApplication() {
    if (!validateBlankStep()) {
      currentStep.value = 2;
      return;
    }
    const projectId = props.projectId;
    if (!projectId) {
      submitError.value = t('application.toast.selectProjectRequired');
      return;
    }
    submitting.value = true;
    submitError.value = '';
    try {
      await applicationApi.create(
        {
          name: blankForm.name,
          code: blankForm.code,
          image_pull_policy: blankForm.image_pull_policy,
          route_managed: blankForm.route_managed,
        },
        { project_id: projectId }
      );
      emit('created');
      router.push('/cd/applications');
    } catch (error) {
      submitError.value = error instanceof Error ? error.message : t('application.toast.createFailed');
    } finally {
      submitting.value = false;
    }
  }

  async function submitImageApplication() {
    if (!validateImageStep()) {
      currentStep.value = 2;
      return;
    }
    const projectId = props.projectId;
    if (!projectId) {
      submitError.value = t('application.toast.selectProjectRequired');
      return;
    }
    submitting.value = true;
    submitError.value = '';
    try {
      const routeDomain = imageForm.routeDomain.trim();
      let app = createdApplication.value;
      if (!app) {
        app = await applicationApi.create(
          {
            name: imageForm.name.trim(),
            code: imageForm.code.trim(),
            image_pull_policy: imageForm.imagePullPolicy,
            route_managed: routeDomain !== '',
          },
          { project_id: projectId }
        );
        createdApplication.value = app;
      }

      if (routeDomain && !app.route_managed) {
        app = await applicationApi.update(app.id, { route_managed: true });
        createdApplication.value = app;
      }

      await upsertFile(app.id, 'docker-compose.yml', composePreview.value);
      if (normalizedEnvVars.value.length > 0) {
        await upsertFile(app.id, '.env', envPreview.value);
      }
      if (routeDomain) {
        await upsertRoute(app.id, routeDomain, containerPort.value);
      }

      emit('created');
      router.push('/cd/applications');
    } catch (error) {
      submitError.value = error instanceof Error ? error.message : t('application.toast.createFailed');
    } finally {
      submitting.value = false;
    }
  }

  async function upsertFile(applicationId: string, path: string, content: string) {
    const files = await applicationApi.listFiles(applicationId);
    const existing = files.find((file) => file.path === path);
    if (existing) {
      await applicationApi.writeFile(applicationId, existing.id, path, content);
      return;
    }
    await applicationApi.createFile(applicationId, path, content);
  }

  async function upsertRoute(applicationId: string, domain: string, port: number) {
    const routes = await applicationApi.listRoutes(applicationId);
    const existing = routes.find((route) => route.service_name === imageCreateServiceName);
    const payload = { service_name: imageCreateServiceName, domain, port };
    if (existing) {
      await applicationApi.updateRoute(applicationId, existing.id, payload);
      return;
    }
    await applicationApi.createRoute(applicationId, payload);
  }

  function goCreatedApplicationDetail() {
    if (!createdApplication.value) {
      return;
    }
    router.push(`/cd/applications/${createdApplication.value.id}`);
  }
</script>
