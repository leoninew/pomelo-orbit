<template>
	<div class="flex flex-col gap-4">
		<div class="flex flex-wrap items-center justify-between gap-3">
			<h1 class="text-xl font-semibold text-foreground flex items-center gap-2">
				{{ application?.name || '应用详情' }}
				<span
					v-if="application"
					class="inline-block rounded-full border px-2.5 py-0.5 text-xs font-medium"
					:class="statusBadgeClass"
				>
					{{ statusText }}
				</span>
			</h1>
			<div class="flex flex-wrap items-center gap-2">
				<button v-if="application" class="app-button-primary h-9 px-3" @click="openEditModal">
					<Pencil class="size-4" />
					编辑
				</button>
				<button
					v-if="application"
					:disabled="operating"
					class="app-button-primary h-9 px-3"
					@click="handleDeploy"
				>
					<Play class="size-4" />
					部署
				</button>
				<button
					v-if="application"
					:disabled="operating"
					class="app-button h-9 px-3"
					@click="handleStop"
				>
					<Square class="size-4" />
					停止
				</button>
				<button
					v-if="application"
					:disabled="operating"
					class="app-button h-9 px-3"
					@click="handleRestart"
				>
					<RotateCcw class="size-4" />
					重启
				</button>
				<button v-if="application" class="app-button h-9 px-3" @click="handleExport">
					<Download class="size-4" />
					导出
				</button>
				<button
					v-if="application"
					:disabled="
						operating || application.status === 'deployed' || application.status === 'deploying'
					"
					class="app-button-danger h-9 px-3"
					@click="openDeleteModal"
				>
					<Trash2 class="size-4" />
					删除
				</button>
				<button class="app-button h-9 px-4" @click="router.push('/cd/applications')">
					<ArrowLeft class="size-4" />
					返回
				</button>
			</div>
		</div>

		<!-- 加载状态 -->
		<AppSpinner v-if="basicInfoLoading" class="py-12" />

		<!-- 内容 -->
		<div v-else-if="application" class="flex flex-col gap-4">
			<!-- 基本信息卡片 -->
			<div class="app-surface">
				<div class="app-section-header">
					<h2 class="font-semibold text-foreground">基本信息</h2>
				</div>
				<dl class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2">
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">应用名称</dt>
						<dd class="text-foreground">{{ application.name }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">应用代码</dt>
						<dd class="text-foreground">{{ application.code }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">状态</dt>
						<dd>
							<span
								class="inline-block rounded-full border px-2.5 py-0.5 text-xs font-medium"
								:class="statusBadgeClass"
							>
								{{ statusText }}
							</span>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">拉取策略</dt>
						<dd class="text-foreground">{{ application.image_pull_policy }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">路由管理</dt>
						<dd class="text-foreground">{{ application.route_managed ? '已启用' : '未启用' }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">部署记录</dt>
						<dd>
							<router-link
								:to="`/cd/deployments?application_id=${application.id}`"
								class="text-primary hover:underline"
							>
								查看所有部署
							</router-link>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">创建时间</dt>
						<dd class="text-muted-foreground">{{ formatTime(application.created_at) }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">更新时间</dt>
						<dd class="text-muted-foreground">{{ formatTime(application.updated_at) }}</dd>
					</div>
				</dl>
			</div>

			<!-- 配置文件 -->
			<div class="app-surface">
				<div class="app-section-header flex items-center justify-between">
					<h2 class="font-semibold text-foreground">配置文件</h2>
					<div class="flex items-center gap-2">
						<button class="app-button-primary h-8 px-3" @click="openAddFileDrawer">
							<Plus class="size-4" />
							添加文件
						</button>
						<button class="app-button h-8 px-3" @click="openComposePreview">
							<Eye class="size-4" />
							预览
						</button>
					</div>
				</div>
				<div class="overflow-x-auto">
					<table class="app-table-detail min-w-[760px]">
						<thead>
							<tr>
								<th>文件路径</th>
								<th>创建时间</th>
								<th>操作</th>
							</tr>
						</thead>
						<tbody>
							<tr v-if="fileListLoading">
								<td colspan="3" class="text-center text-muted-foreground">
									<AppSpinner />
								</td>
							</tr>
							<tr v-else-if="files.length === 0">
								<td colspan="3" class="text-center text-muted-foreground">暂无配置文件</td>
							</tr>
							<tr v-for="file in files" :key="file.id">
								<td class="max-w-md truncate text-foreground" :title="file.path">
									{{ file.path }}
								</td>
								<td class="text-muted-foreground">{{ formatTime(file.created_at) }}</td>
								<td>
									<div class="flex items-center gap-3">
										<button class="app-link" @click="openFileDrawer(file.id, false)">查看</button>
										<button class="app-link" @click="openFileDrawer(file.id, true)">编辑</button>
										<button class="app-link-danger" @click="confirmDeleteFile(file.id)">
											删除
										</button>
									</div>
								</td>
							</tr>
						</tbody>
					</table>
				</div>
			</div>

			<!-- 镜像配置 -->
			<div class="app-surface">
				<div class="app-section-header">
					<h2 class="font-semibold text-foreground">镜像配置</h2>
				</div>
				<div class="overflow-x-auto">
					<table class="app-table-detail min-w-[960px]">
						<thead>
							<tr>
								<th>服务</th>
								<th>默认域名</th>
								<th>默认端口</th>
								<th>基础镜像</th>
								<th>当前镜像</th>
								<th>已覆盖</th>
								<th>更新时间</th>
								<th>操作</th>
							</tr>
						</thead>
						<tbody>
							<tr v-if="serviceConfigListLoading">
								<td colspan="8" class="text-center text-muted-foreground">
									<AppSpinner />
								</td>
							</tr>
							<tr v-else-if="serviceConfigError">
								<td colspan="8" class="text-center text-destructive">
									{{ serviceConfigError }}
								</td>
							</tr>
							<tr v-else-if="serviceConfigs.length === 0">
								<td colspan="8" class="text-center text-muted-foreground">暂无镜像配置</td>
							</tr>
							<tr v-for="config in serviceConfigs" :key="config.service_name">
								<td class="text-foreground">{{ config.service_name }}</td>
								<td class="text-muted-foreground">{{ config.default_domain }}</td>
								<td class="text-muted-foreground">{{ config.default_port }}</td>
								<td
									class="max-w-xs truncate text-muted-foreground"
									:title="config.base_image ?? ''"
								>
									{{ config.base_image || '—' }}
								</td>
								<td
									class="max-w-xs truncate text-foreground"
									:title="getServiceDisplayImage(config)"
								>
									{{ getServiceDisplayImage(config) || '—' }}
								</td>
								<td>
									<span
										v-if="isServiceOverridden(config)"
										class="inline-block rounded-full border border-primary/30 bg-primary/10 px-2 py-0.5 text-xs font-medium text-primary"
									>
										已覆盖
									</span>
									<span v-else class="text-muted-foreground">—</span>
								</td>
								<td class="text-muted-foreground">
									{{ config.updated_at ? formatTime(config.updated_at) : '—' }}
								</td>
								<td>
									<div class="flex items-center gap-3">
										<button class="app-link" @click="openEditServiceConfigModal(config)">
											编辑
										</button>
										<button
											v-if="canResetServiceConfig(config)"
											class="app-link-danger"
											@click="confirmResetServiceConfig(config)"
										>
											重置
										</button>
										<span v-else class="text-muted-foreground">—</span>
									</div>
								</td>
							</tr>
						</tbody>
					</table>
				</div>
			</div>

			<!-- 路由配置 -->
			<div class="app-surface">
				<div class="app-section-header flex items-center justify-between">
					<h2 class="font-semibold text-foreground">路由配置</h2>
					<button
						class="app-button-primary h-8 px-3"
						:disabled="!application.route_managed"
						:title="!application.route_managed ? '请先在基本信息中启用路由管理' : undefined"
						@click="openAddRouteModal"
					>
						<Plus class="size-4" />
						添加路由
					</button>
				</div>
				<div v-if="!application.route_managed" class="px-5 pt-4">
					<div class="app-tip">请先在基本信息中启用路由管理，再添加或编辑路由配置。</div>
				</div>
				<div class="overflow-x-auto">
					<table class="app-table-detail min-w-[760px]">
						<thead>
							<tr>
								<th>域名</th>
								<th>服务</th>
								<th>端口</th>
								<th>创建时间</th>
								<th>操作</th>
							</tr>
						</thead>
						<tbody>
							<tr v-if="routeListLoading">
								<td colspan="5" class="text-center text-muted-foreground">
									<AppSpinner />
								</td>
							</tr>
							<tr v-else-if="appRoutes.length === 0">
								<td colspan="5" class="text-center text-muted-foreground">暂无路由配置</td>
							</tr>
							<tr v-for="r in appRoutes" :key="r.id">
								<td class="text-foreground">{{ r.domain }}</td>
								<td class="text-muted-foreground">{{ r.service_name }}</td>
								<td class="text-muted-foreground">{{ r.port }}</td>
								<td class="text-muted-foreground">{{ formatTime(r.created_at) }}</td>
								<td>
									<div class="flex items-center gap-3">
										<button
											class="app-link"
											:disabled="!application.route_managed"
											@click="openEditRouteModal(r)"
										>
											编辑
										</button>
										<button class="app-link-danger" @click="confirmDeleteRoute(r.id)">删除</button>
									</div>
								</td>
							</tr>
						</tbody>
					</table>
				</div>
			</div>
		</div>

		<AppDrawer
			:open="fileDrawerVisible"
			:title="fileDrawerTitle"
			width-class="w-[min(960px,100vw)]"
			body-class="min-h-0 flex-1 overflow-hidden p-0"
			@update:open="handleFileDrawerOpenChange"
		>
			<div class="flex h-full flex-col gap-4 p-6">
				<div class="space-y-1.5">
					<label class="app-field-label block">文件路径</label>
					<input
						v-model="currentFilePath"
						type="text"
						:disabled="!isEditingInDrawer"
						placeholder="例如: docker-compose.yml"
						class="app-input"
					/>
				</div>
				<div class="min-h-0 flex-1 space-y-1.5">
					<label class="app-field-label block">文件内容</label>
					<div
						v-if="fileContentLoading"
						class="flex h-[calc(100%-1.75rem)] items-center justify-center rounded-md border border-border bg-muted/30"
					>
						<AppSpinner />
					</div>
					<textarea
						v-else
						v-model="currentFileContent"
						:readonly="!isEditingInDrawer"
						class="app-textarea h-[calc(100%-1.75rem)] resize-none font-mono"
					></textarea>
				</div>
			</div>
			<template #footer>
				<template v-if="!isEditingInDrawer && currentFileId">
					<button class="app-button" @click="handleDrawerClose">关闭</button>
					<button
						class="app-button-primary"
						:disabled="fileContentLoading"
						@click="isEditingInDrawer = true"
					>
						编辑
					</button>
				</template>
				<template v-else>
					<button class="app-button" @click="handleDrawerClose">取消</button>
					<button
						:disabled="fileContentLoading"
						class="app-button-primary"
						@click="saveCurrentFile"
					>
						保存
					</button>
				</template>
			</template>
		</AppDrawer>

		<AppDialog v-model:open="isEditDialogOpen" title="编辑应用">
			<div>
				<label class="app-field-label mb-1.5 block">
					应用名称
					<span class="text-destructive">*</span>
				</label>
				<input
					v-model="editForm.name"
					type="text"
					class="app-input"
					:class="editErrors.name ? 'app-input-error' : ''"
				/>
				<p v-if="editErrors.name" class="app-field-error mt-1 text-xs">{{ editErrors.name }}</p>
			</div>
			<div>
				<label class="app-field-label mb-1.5 block">应用代码</label>
				<input v-model="editForm.code" type="text" disabled class="app-input" />
			</div>
			<div>
				<label class="app-field-label mb-1.5 block">镜像拉取策略</label>
				<SelectControl v-model="editForm.image_pull_policy" :options="imagePullPolicyOptions" />
			</div>
			<div class="space-y-1.5">
				<label class="flex items-center gap-2">
					<input v-model="editForm.route_managed" type="checkbox" class="app-checkbox" />
					<span class="text-sm font-medium text-foreground">启用路由管理</span>
				</label>
				<p class="app-field-hint ml-6">启用后，部署时将自动生成 Traefik 路由配置。</p>
			</div>
			<template #footer>
				<button class="app-button" @click="isEditDialogOpen = false">取消</button>
				<button :disabled="operating" class="app-button-primary" @click="handleEditOk">保存</button>
			</template>
		</AppDialog>

		<AppDialog
			v-model:open="isDeleteDialogOpen"
			title="确认删除"
			:description="`确定要删除应用「${application?.name || '当前应用'}」吗？此操作不可恢复。`"
			width-class="w-[min(420px,calc(100vw-32px))]"
		>
			<label class="flex items-center gap-2">
				<input v-model="deleteDir" type="checkbox" class="app-checkbox" />
				<span class="text-sm text-foreground">
					同时删除应用工作目录（data/apps/{{ application?.code || '-' }}）
				</span>
			</label>
			<template #footer>
				<button class="app-button" @click="isDeleteDialogOpen = false">取消</button>
				<button :disabled="operating" class="app-button-destructive" @click="handleDeleteOk">
					删除
				</button>
			</template>
		</AppDialog>

		<AppDialog
			v-model:open="isDeleteFileDialogOpen"
			title="确认删除"
			description="确定要删除这个配置文件吗？此操作无法撤销。"
			width-class="w-[min(420px,calc(100vw-32px))]"
			body-class="hidden"
		>
			<template #footer>
				<button class="app-button" @click="isDeleteFileDialogOpen = false">取消</button>
				<button
					:disabled="fileListLoading"
					class="app-button-destructive"
					@click="executeDeleteFile"
				>
					删除
				</button>
			</template>
		</AppDialog>

		<AppDialog v-model:open="isServiceConfigDialogOpen" title="编辑服务镜像">
			<div>
				<label class="app-field-label mb-1.5 block">服务</label>
				<input :value="selectedServiceName" type="text" disabled class="app-input" />
			</div>
			<div>
				<label class="app-field-label mb-1.5 block">镜像</label>
				<input
					v-model="serviceConfigForm.image"
					type="text"
					class="app-input"
					placeholder="例如: nginx:1.28"
				/>
				<p class="app-field-hint mt-1.5">留空会移除单独覆盖，回退到 compose 中的 image。</p>
			</div>
			<template #footer>
				<button class="app-button" @click="isServiceConfigDialogOpen = false">取消</button>
				<button
					:disabled="!serviceConfigDirty || serviceConfigSaving"
					class="app-button-primary"
					@click="saveServiceConfig"
				>
					保存
				</button>
			</template>
		</AppDialog>

		<AppDialog
			v-model:open="isDeleteServiceConfigDialogOpen"
			title="确认重置"
			:description="`确定要重置「${pendingDeleteServiceName || '当前服务'}」的镜像覆盖并回退到 compose 原值吗？`"
			width-class="w-[min(420px,calc(100vw-32px))]"
			body-class="hidden"
		>
			<template #footer>
				<button class="app-button" @click="cancelResetServiceConfig">取消</button>
				<button
					:disabled="serviceConfigSaving"
					class="app-button-destructive"
					@click="executeResetServiceConfig"
				>
					重置
				</button>
			</template>
		</AppDialog>

		<AppDialog v-model:open="isRouteDialogOpen" :title="editingRouteId ? '编辑路由' : '添加路由'">
			<div>
				<label class="app-field-label mb-1.5 block">
					选择服务
					<span class="text-destructive">*</span>
				</label>
				<ComboboxSelect
					:model-value="routeForm.service_name"
					:options="composeServiceOptions"
					:invalid="Boolean(routeFormErrors.service_name)"
					:portal="false"
					placeholder="搜索服务..."
					empty-text="未找到服务"
					@update:model-value="handleRouteServiceChange"
				/>
				<p v-if="routeFormErrors.service_name" class="app-field-error mt-1 text-xs">
					{{ routeFormErrors.service_name }}
				</p>
			</div>
			<div>
				<label class="app-field-label mb-1.5 block">
					域名
					<span class="text-destructive">*</span>
				</label>
				<input
					v-model="routeForm.domain"
					type="text"
					placeholder="example.com"
					class="app-input"
					:class="routeFormErrors.domain ? 'app-input-error' : ''"
				/>
				<p v-if="routeFormErrors.domain" class="app-field-error mt-1 text-xs">
					{{ routeFormErrors.domain }}
				</p>
			</div>
			<div>
				<label class="app-field-label mb-1.5 block">
					容器端口
					<span class="text-destructive">*</span>
				</label>
				<input
					v-model.number="routeForm.port"
					type="number"
					min="1"
					max="65535"
					placeholder="例如: 80"
					class="app-input"
					:class="routeFormErrors.port ? 'app-input-error' : ''"
				/>
				<p v-if="routeFormErrors.port" class="app-field-error mt-1 text-xs">
					{{ routeFormErrors.port }}
				</p>
			</div>
			<template #footer>
				<button class="app-button" @click="isRouteDialogOpen = false">取消</button>
				<button :disabled="routeLoading" class="app-button-primary" @click="handleRouteOk">
					{{ editingRouteId ? '保存' : '添加' }}
				</button>
			</template>
		</AppDialog>

		<AppDialog
			v-model:open="isDeleteRouteDialogOpen"
			title="确认删除"
			description="确定要删除这个路由配置吗？此操作无法撤销。"
			width-class="w-[min(420px,calc(100vw-32px))]"
			body-class="hidden"
		>
			<template #footer>
				<button class="app-button" @click="isDeleteRouteDialogOpen = false">取消</button>
				<button :disabled="routeLoading" class="app-button-destructive" @click="executeDeleteRoute">
					删除
				</button>
			</template>
		</AppDialog>

		<AppDrawer
			:open="composePreviewDrawerOpen"
			title="docker-compose 预览"
			width-class="w-[min(960px,100vw)]"
			@update:open="handleComposePreviewDrawerOpenChange"
		>
			<div class="space-y-3">
				<p class="text-sm text-muted-foreground">
					与部署时写入的 docker-compose.yml 一致：模板渲染、服务镜像覆盖、路由托管 labels。
				</p>
				<div v-if="composePreviewLoading" class="flex justify-center py-12">
					<AppSpinner />
				</div>
				<div
					v-else-if="composePreviewError"
					class="rounded-md border border-destructive/30 bg-destructive/5 px-4 py-3 text-sm text-destructive"
				>
					{{ composePreviewError }}
				</div>
				<div
					v-else
					class="rounded-md border border-border bg-muted/30 px-4 py-3 font-mono text-xs leading-relaxed whitespace-pre-wrap break-words overflow-auto"
					style="max-height: calc(100vh - 200px)"
				>
					{{ composePreviewYaml }}
				</div>
			</div>
			<template #footer>
				<button class="app-button" @click="composePreviewDrawerOpen = false">关闭</button>
			</template>
		</AppDrawer>
	</div>
</template>

<script setup lang="ts">
	import {
		ArrowLeft,
		Download,
		Eye,
		Pencil,
		Play,
		Plus,
		RotateCcw,
		Square,
		Trash2,
	} from 'lucide-vue-next';
	import { computed, onMounted, reactive, ref } from 'vue';
	import { useRoute, useRouter } from 'vue-router';
	import { applicationApi } from '@/api/cd/application';
	import { deploymentApi } from '@/api/cd/deployments';
	import AppDialog from '@/components/AppDialog.vue';
	import AppSpinner from '@/components/AppSpinner.vue';
	import AppDrawer from '@/components/AppDrawer.vue';
	import ComboboxSelect, { type ComboboxOptionValue } from '@/components/ComboboxSelect.vue';
	import SelectControl from '@/components/SelectControl.vue';
	import { useStatusAsync } from '@/composables/useStatusAsync';
	import { useToast } from '@/composables/useToast';
	import type {
		Application,
		ApplicationRoute,
		ApplicationServiceConfig,
		ComposeServiceResp,
		ConfigFile,
	} from '@/types/cd/application';
	import { delayAsync, formatTime } from '@/utils/time';

	const route = useRoute();
	const router = useRouter();
	const applicationId = route.params.id as string;
	const toast = useToast();

	const { loading: basicInfoLoading, execute: executeBasicInfo } = useStatusAsync();
	const { loading: operating, execute: executeOp } = useStatusAsync();
	const { loading: fileListLoading, execute: executeFileList } = useStatusAsync();
	const { loading: fileContentLoading, execute: executeFileContent } = useStatusAsync();
	const { loading: routeListLoading, execute: executeRouteList } = useStatusAsync();
	const { loading: routeLoading, execute: executeRoute } = useStatusAsync();
	const { loading: serviceConfigListLoading, execute: executeServiceConfigList } = useStatusAsync();
	const { loading: serviceConfigSaving, execute: executeServiceConfigSave } = useStatusAsync();
	const { loading: composePreviewLoading, execute: executeComposePreview } = useStatusAsync();

	const application = ref<Application>();
	const files = ref<ConfigFile[]>([]);
	const appRoutes = ref<ApplicationRoute[]>([]);
	const composeServices = ref<ComposeServiceResp[]>([]);
	const serviceConfigs = ref<ApplicationServiceConfig[]>([]);
	const selectedServiceName = ref('');
	const serviceConfigError = ref('');

	const isEditDialogOpen = ref(false);
	const isDeleteDialogOpen = ref(false);
	const isDeleteFileDialogOpen = ref(false);
	const isServiceConfigDialogOpen = ref(false);
	const isDeleteServiceConfigDialogOpen = ref(false);
	const isRouteDialogOpen = ref(false);
	const isDeleteRouteDialogOpen = ref(false);
	const composePreviewDrawerOpen = ref(false);
	const pendingDeleteFileId = ref('');
	const pendingDeleteServiceName = ref('');
	const pendingDeleteRouteId = ref('');
	const editingRouteId = ref('');
	const deleteDir = ref(false);
	const composePreviewYaml = ref('');
	const composePreviewError = ref('');

	const routeForm = reactive({ service_name: '', domain: '', port: 80 });
	const routeFormErrors = reactive({ service_name: '', domain: '', port: '' });
	const serviceConfigForm = reactive({ image: '' });

	const fileDrawerVisible = ref(false);
	const currentFileId = ref('');
	const currentFilePath = ref('');
	const currentFileContent = ref('');
	const isEditingInDrawer = ref(false);

	const editForm = reactive({
		name: '',
		code: '',
		image_pull_policy: 'missing',
		route_managed: false,
	});
	const editErrors = reactive({ name: '' });
	const imagePullPolicyOptions = [
		{ value: 'missing', label: '缺失时拉取 (missing)' },
		{ value: 'always', label: '总是拉取 (always)' },
		{ value: 'never', label: '从不拉取 (never)' },
	];

	const activeServiceConfig = computed(() =>
		serviceConfigs.value.find((item) => item.service_name === selectedServiceName.value)
	);
	const currentServiceImage = computed(() =>
		activeServiceConfig.value ? getServiceDisplayImage(activeServiceConfig.value) : ''
	);

	const composeServiceOptions = computed(() =>
		composeServices.value.map((service) => ({
			value: service.service_name,
			label: service.service_name,
			description: `${service.default_domain}:${service.default_port}`,
		}))
	);
	const fileDrawerTitle = computed(() => {
		if (!currentFileId.value) {
			return '添加文件';
		}
		const action = isEditingInDrawer.value ? '编辑文件' : '查看文件';
		return currentFilePath.value ? `${action}: ${currentFilePath.value}` : action;
	});
	const serviceConfigDirty = computed(
		() => serviceConfigForm.image.trim() !== currentServiceImage.value.trim()
	);

	const statusBadgeClass = computed(() => {
		const status = application.value?.status;
		if (!status) {
			return 'bg-muted/50 text-muted-foreground';
		}
		const map: Record<string, string> = {
			deployed: 'bg-green-50 text-green-700 border-green-200',
			deploy_failed: 'bg-red-50 text-red-700 border-red-200',
			deploying: 'bg-blue-50 text-blue-700 border-blue-200',
			undeployed: 'bg-muted/50 text-muted-foreground',
		};
		return map[status] || 'bg-muted/50 text-muted-foreground';
	});

	const statusText = computed(() => {
		const status = application.value?.status;
		if (!status) {
			return '';
		}
		const map: Record<string, string> = {
			deployed: '已部署',
			deploy_failed: '部署失败',
			deploying: '部署中',
			undeployed: '未部署',
		};
		return map[status] || status;
	});

	async function fetchApplication() {
		try {
			await executeBasicInfo(async () => {
				const data = await applicationApi.get(applicationId);
				application.value = data;
				Object.assign(editForm, {
					name: data.name,
					code: data.code,
					image_pull_policy: data.image_pull_policy,
					route_managed: data.route_managed,
				});
			});
			if (application.value?.status === 'deploying') {
				pollActiveDeployment();
			}
		} catch {
			toast.error('获取应用信息失败');
			router.push('/cd/applications');
		}
	}

	async function pollActiveDeployment() {
		try {
			const resp = await deploymentApi.list({
				application_id: applicationId,
				per_page: 1,
			});
			const latest = resp.items[0];
			if (!latest) {
				return;
			}
			while (true) {
				await delayAsync(3000);
				try {
					const detail = await deploymentApi.get(latest.id);
					if (['ran_to_completion', 'faulted', 'canceled'].includes(detail.status)) {
						if (application.value) {
							application.value.status =
								detail.status === 'ran_to_completion' ? 'deployed' : 'deploy_failed';
						}
						break;
					}
				} catch {
					break;
				}
			}
		} catch {
			/* silent */
		}
	}

	async function handleDeploy() {
		try {
			await executeOp(async () => {
				const res = await applicationApi.deploy(applicationId);
				toast.success('部署已触发');
				router.push({
					path: `/cd/deployments/${res.deployment_id}`,
					query: { from: 'application' },
				});
			});
		} catch (error) {
			toast.error(error instanceof Error ? error.message : '触发部署失败');
		}
	}

	async function handleStop() {
		try {
			await executeOp(async () => {
				const res = await applicationApi.stop(applicationId);
				toast.success('停止操作已提交');
				const maxAttempts = 20; // 最多轮询 20 次（60 秒）
				let attempts = 0;
				while (attempts < maxAttempts) {
					await delayAsync(3000);
					attempts++;
					try {
						const detail = await deploymentApi.get(res.deployment_id);
						if (['ran_to_completion', 'faulted', 'canceled'].includes(detail.status)) {
							await fetchApplication(); // 重新加载应用状态
							break;
						}
					} catch {
						break;
					}
				}
			});
		} catch (error) {
			toast.error(error instanceof Error ? error.message : '停止失败');
		}
	}

	async function handleRestart() {
		try {
			await executeOp(async () => {
				const res = await applicationApi.restart(applicationId);
				toast.success('重启操作已提交');
				router.push({
					path: `/cd/deployments/${res.deployment_id}`,
					query: { from: 'application' },
				});
			});
		} catch (error) {
			toast.error(error instanceof Error ? error.message : '重启失败');
		}
	}

	async function handleExport() {
		try {
			const data = await applicationApi.exportApplication(applicationId);
			const blob = new Blob([JSON.stringify(data, null, 2)], {
				type: 'application/json',
			});
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `${data.code || 'application'}.json`;
			a.click();
			URL.revokeObjectURL(url);
			toast.success('导出成功');
		} catch (error) {
			toast.error(error instanceof Error ? error.message : '导出失败');
		}
	}

	function openEditModal() {
		editErrors.name = '';
		if (application.value) {
			Object.assign(editForm, {
				name: application.value.name,
				code: application.value.code,
				image_pull_policy: application.value.image_pull_policy,
				route_managed: application.value.route_managed,
			});
		}
		isEditDialogOpen.value = true;
	}

	async function handleEditOk() {
		editErrors.name = editForm.name.trim() ? '' : '请输入应用名称';
		if (editErrors.name) {
			return;
		}
		try {
			await executeOp(async () => {
				await applicationApi.update(applicationId, {
					name: editForm.name,
					image_pull_policy: editForm.image_pull_policy,
					route_managed: editForm.route_managed,
				});
				toast.success('更新成功');
				isEditDialogOpen.value = false;
				await fetchApplication();
				if (application.value?.route_managed) {
					await loadRoutes();
				} else {
					appRoutes.value = [];
				}
			});
		} catch (error) {
			toast.error(error instanceof Error ? error.message : '更新失败');
		}
	}

	function openDeleteModal() {
		deleteDir.value = false;
		isDeleteDialogOpen.value = true;
	}

	async function handleDeleteOk() {
		try {
			await executeOp(async () => {
				await applicationApi.delete(applicationId, deleteDir.value);
				toast.success('删除成功');
				router.push('/cd/applications');
			});
		} catch (error) {
			toast.error(error instanceof Error ? error.message : '删除失败');
		}
	}

	async function loadFiles() {
		try {
			await executeFileList(async () => {
				files.value = await applicationApi.listFiles(applicationId);
			});
		} catch {
			toast.error('加载配置文件失败');
		}
	}

	async function loadServiceConfigs() {
		serviceConfigError.value = '';
		try {
			await executeServiceConfigList(async () => {
				serviceConfigs.value = await applicationApi.listServiceConfigs(applicationId);
			});
		} catch (error) {
			serviceConfigs.value = [];
			serviceConfigError.value = error instanceof Error ? error.message : '加载 service 配置失败';
		}
	}

	function getServiceDisplayImage(serviceConfig: ApplicationServiceConfig) {
		return serviceConfig.image?.trim() || serviceConfig.base_image || '';
	}

	function isServiceOverridden(serviceConfig: ApplicationServiceConfig) {
		const overrideImage = serviceConfig.image?.trim() || '';
		const baseImage = serviceConfig.base_image?.trim() || '';
		return Boolean(overrideImage) && overrideImage !== baseImage;
	}

	function canResetServiceConfig(serviceConfig: ApplicationServiceConfig) {
		return Boolean(serviceConfig.image?.trim());
	}

	function openEditServiceConfigModal(serviceConfig: ApplicationServiceConfig) {
		selectedServiceName.value = serviceConfig.service_name;
		serviceConfigForm.image = getServiceDisplayImage(serviceConfig);
		isServiceConfigDialogOpen.value = true;
	}

	function confirmResetServiceConfig(serviceConfig: ApplicationServiceConfig) {
		if (!canResetServiceConfig(serviceConfig)) {
			return;
		}
		pendingDeleteServiceName.value = serviceConfig.service_name;
		isDeleteServiceConfigDialogOpen.value = true;
	}

	function cancelResetServiceConfig() {
		pendingDeleteServiceName.value = '';
		isDeleteServiceConfigDialogOpen.value = false;
	}

	async function executeResetServiceConfig() {
		if (!pendingDeleteServiceName.value) {
			return;
		}
		try {
			await executeServiceConfigSave(async () => {
				const saved = await applicationApi.updateServiceConfig(
					applicationId,
					pendingDeleteServiceName.value,
					null
				);
				const idx = serviceConfigs.value.findIndex(
					(item) => item.service_name === saved.service_name
				);
				if (idx >= 0) {
					serviceConfigs.value[idx] = saved;
				} else {
					serviceConfigs.value.push(saved);
				}
				if (selectedServiceName.value === saved.service_name) {
					serviceConfigForm.image = getServiceDisplayImage(saved);
					isServiceConfigDialogOpen.value = false;
				}
				isDeleteServiceConfigDialogOpen.value = false;
				pendingDeleteServiceName.value = '';
				toast.success('服务镜像已重置');
			});
		} catch (error) {
			toast.error(error instanceof Error ? error.message : '重置服务镜像失败');
		}
	}

	async function saveServiceConfig() {
		const active = activeServiceConfig.value;
		if (!active || !serviceConfigDirty.value) {
			return;
		}
		try {
			await executeServiceConfigSave(async () => {
				const saved = await applicationApi.updateServiceConfig(
					applicationId,
					active.service_name,
					serviceConfigForm.image
				);
				const idx = serviceConfigs.value.findIndex(
					(item) => item.service_name === saved.service_name
				);
				if (idx >= 0) {
					serviceConfigs.value[idx] = saved;
				} else {
					serviceConfigs.value.push(saved);
				}
				selectedServiceName.value = saved.service_name;
				serviceConfigForm.image = getServiceDisplayImage(saved);
				isServiceConfigDialogOpen.value = false;
				toast.success('服务镜像保存成功');
			});
		} catch (error) {
			toast.error(error instanceof Error ? error.message : '保存服务镜像失败');
		}
	}

	async function openFileDrawer(fileId: string, isEdit = false) {
		currentFileId.value = fileId;
		isEditingInDrawer.value = isEdit;
		currentFileContent.value = '';
		currentFilePath.value = '';
		fileDrawerVisible.value = true;
		try {
			await executeFileContent(async () => {
				const result = await applicationApi.readFile(applicationId, fileId);
				currentFileContent.value = result.content ?? '';
				currentFilePath.value = result.path || '';
			});
		} catch {
			toast.error('加载文件内容失败');
		}
	}

	function openAddFileDrawer() {
		currentFileId.value = '';
		currentFilePath.value = '';
		currentFileContent.value = '';
		isEditingInDrawer.value = true;
		fileDrawerVisible.value = true;
	}

	function handleDrawerClose() {
		fileDrawerVisible.value = false;
		isEditingInDrawer.value = false;
	}

	function handleFileDrawerOpenChange(open: boolean) {
		fileDrawerVisible.value = open;
		if (!open) {
			isEditingInDrawer.value = false;
		}
	}

	async function saveCurrentFile() {
		if (!currentFilePath.value.trim()) {
			toast.error('请输入文件路径');
			return;
		}
		const lowerPath = currentFilePath.value.toLowerCase();
		const content =
			lowerPath.endsWith('.sh') || lowerPath.endsWith('.bash')
				? currentFileContent.value.replace(/\r\n/g, '\n')
				: currentFileContent.value;
		try {
			await executeFileContent(async () => {
				if (currentFileId.value) {
					const updated = await applicationApi.writeFile(
						applicationId,
						currentFileId.value,
						currentFilePath.value,
						content
					);
					const idx = files.value.findIndex((f) => f.id === currentFileId.value);
					if (idx >= 0) {
						files.value[idx] = updated;
					}
					toast.success('保存成功');
				} else {
					await applicationApi.createFile(applicationId, currentFilePath.value, content);
					toast.success('添加成功');
				}
				fileDrawerVisible.value = false;
				isEditingInDrawer.value = false;
				await loadFiles();
				await loadServiceConfigs();
			});
		} catch (error) {
			toast.error(error instanceof Error ? error.message : '保存失败');
		}
	}

	function confirmDeleteFile(fileId: string) {
		pendingDeleteFileId.value = fileId;
		isDeleteFileDialogOpen.value = true;
	}

	async function executeDeleteFile() {
		try {
			await executeFileList(async () => {
				await applicationApi.deleteFile(applicationId, pendingDeleteFileId.value);
				toast.success('删除成功');
				isDeleteFileDialogOpen.value = false;
				await loadFiles();
				await loadServiceConfigs();
			});
		} catch {
			toast.error('删除失败');
		}
	}

	async function openComposePreview() {
		composePreviewYaml.value = '';
		composePreviewError.value = '';
		composePreviewDrawerOpen.value = true;
		try {
			await executeComposePreview(async () => {
				const { compose_yaml } = await applicationApi.previewCompose(applicationId);
				composePreviewYaml.value = compose_yaml;
			});
		} catch (error) {
			composePreviewError.value = error instanceof Error ? error.message : '加载预览失败';
		}
	}

	function handleComposePreviewDrawerOpenChange(open: boolean) {
		composePreviewDrawerOpen.value = open;
		if (!open) {
			composePreviewYaml.value = '';
			composePreviewError.value = '';
		}
	}

	// ── Route management ──

	async function loadRoutes() {
		try {
			await executeRouteList(async () => {
				appRoutes.value = await applicationApi.listRoutes(applicationId);
			});
		} catch {
			toast.error('加载路由配置失败');
		}
	}

	async function loadComposeServices() {
		composeServices.value = [];
		try {
			composeServices.value = await applicationApi.listComposeServices(applicationId);
		} catch (error) {
			toast.error(
				error instanceof Error ? error.message : '解析 docker-compose 失败，无法配置路由'
			);
		}
	}

	function handleRouteServiceChange(value: ComboboxOptionValue) {
		const serviceName = String(value || '');
		const service = composeServices.value.find((item) => item.service_name === serviceName);
		if (service) {
			routeForm.service_name = service.service_name;
			routeForm.domain = service.default_domain;
			routeForm.port = service.default_port;
			routeFormErrors.service_name = '';
		} else {
			routeForm.service_name = '';
		}
	}

	async function openAddRouteModal() {
		if (!application.value?.route_managed) {
			return;
		}
		editingRouteId.value = '';
		Object.assign(routeForm, { service_name: '', domain: '', port: 80 });
		Object.assign(routeFormErrors, { service_name: '', domain: '', port: '' });
		await loadComposeServices();
		if (composeServices.value.length === 0) {
			return;
		}
		isRouteDialogOpen.value = true;
	}

	async function openEditRouteModal(r: ApplicationRoute) {
		if (!application.value?.route_managed) {
			return;
		}
		editingRouteId.value = r.id;
		Object.assign(routeForm, { service_name: r.service_name, domain: r.domain, port: r.port });
		Object.assign(routeFormErrors, { service_name: '', domain: '', port: '' });
		await loadComposeServices();
		if (composeServices.value.length === 0) {
			return;
		}
		isRouteDialogOpen.value = true;
	}

	function validateRouteForm() {
		routeFormErrors.service_name = routeForm.service_name ? '' : '请选择 service';
		routeFormErrors.domain = routeForm.domain.trim() ? '' : '请输入域名';
		routeFormErrors.port = routeForm.port >= 1 && routeForm.port <= 65535 ? '' : '端口范围 1-65535';
		return !routeFormErrors.service_name && !routeFormErrors.domain && !routeFormErrors.port;
	}

	async function handleRouteOk() {
		if (!validateRouteForm()) {
			return;
		}
		try {
			await executeRoute(async () => {
				const data = {
					service_name: routeForm.service_name,
					domain: routeForm.domain,
					port: routeForm.port,
				};
				if (editingRouteId.value) {
					await applicationApi.updateRoute(applicationId, editingRouteId.value, data);
				} else {
					await applicationApi.createRoute(applicationId, data);
				}
				toast.success('保存成功');
				isRouteDialogOpen.value = false;
				await loadRoutes();
			});
		} catch (error) {
			toast.error(error instanceof Error ? error.message : '保存失败');
		}
	}

	function confirmDeleteRoute(routeId: string) {
		pendingDeleteRouteId.value = routeId;
		isDeleteRouteDialogOpen.value = true;
	}

	async function executeDeleteRoute() {
		try {
			await executeRoute(async () => {
				await applicationApi.deleteRoute(applicationId, pendingDeleteRouteId.value);
				toast.success('删除成功');
				isDeleteRouteDialogOpen.value = false;
				await loadRoutes();
			});
		} catch {
			toast.error('删除失败');
		}
	}

	onMounted(async () => {
		await fetchApplication();
		await loadFiles();
		await loadServiceConfigs();
		if (application.value?.route_managed) {
			await loadRoutes();
		}
	});
</script>
