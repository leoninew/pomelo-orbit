<template>
	<div class="flex flex-col gap-4">
		<!-- Page header -->
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold flex items-center gap-2">
				{{ application?.name ?? '应用详情' }}
				<span v-if="application" class="badge badge-sm" :class="appBadgeClass(application.status)">
					{{ appStatusLabel(application.status) }}
				</span>
			</h1>
			<button class="btn btn-sm btn-ghost gap-1" @click="$router.push('/cd/applications')">
				<ArrowLeft class="size-4" />
				返回
			</button>
		</div>

		<!-- Basic info card -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-5">
				<div class="flex items-center justify-between mb-4">
					<h2 class="font-semibold">基本信息</h2>
					<div v-if="application" class="flex items-center gap-2 flex-wrap">
						<!-- Deploy with optional env dropdown -->
						<div v-if="envs.length > 1" class="dropdown dropdown-end">
							<button tabindex="0" class="btn btn-sm btn-primary gap-1" :disabled="operating">
								<Rocket class="size-3.5" />
								部署
								<ChevronDown class="size-3" />
							</button>
							<ul
								tabindex="0"
								class="dropdown-content menu bg-base-100 rounded-box shadow-lg border border-base-200 w-40 mt-1 p-1 z-50"
							>
								<li v-for="env in envs.filter((e) => e !== '.env')" :key="env">
									<button @click="handleDeployWithEnv(env)">{{ env }}</button>
								</li>
							</ul>
						</div>
						<button
							v-else
							class="btn btn-sm btn-primary gap-1"
							:disabled="operating"
							@click="handleDeploy"
						>
							<Rocket class="size-3.5" />
							部署
						</button>
						<button class="btn btn-sm btn-ghost" :disabled="operating" @click="handleStop">
							停止
						</button>
						<button class="btn btn-sm btn-ghost" :disabled="operating" @click="handleRestart">
							重启
						</button>
						<button class="btn btn-sm btn-ghost" :disabled="operating" @click="openEditModal">
							编辑
						</button>
						<button class="btn btn-sm btn-ghost gap-1" @click="handleExport">
							<Download class="size-3.5" />
							导出
						</button>
						<button
							class="btn btn-sm btn-error btn-ghost"
							:disabled="
								operating || application.status === 'deployed' || application.status === 'deploying'
							"
							@click="openDeleteModal"
						>
							删除
						</button>
					</div>
				</div>

				<div v-if="basicInfoLoading" class="flex justify-center py-8">
					<span class="loading loading-spinner loading-md text-primary" />
				</div>
				<dl v-else-if="application" class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-3 text-sm">
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">应用编码</dt>
						<dd>
							<code class="text-xs bg-base-200 px-1.5 py-0.5 rounded">{{ application.code }}</code>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">拉取策略</dt>
						<dd>{{ application.image_pull_policy }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">路由托管</dt>
						<dd>
							<span
								v-if="application.route_managed"
								class="badge badge-sm badge-success badge-outline"
							>
								已启用
							</span>
							<span v-else class="badge badge-sm badge-ghost">未启用</span>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">创建时间</dt>
						<dd class="text-base-content/60">{{ formatTime(application.created_at) }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">部署记录</dt>
						<dd>
							<router-link
								:to="`/cd/deployments?application_id=${application.id}`"
								class="link link-primary"
							>
								查看所有部署
							</router-link>
						</dd>
					</div>
				</dl>
			</div>
		</div>

		<!-- Config files card -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-5">
				<div class="flex items-center justify-between mb-4">
					<h2 class="font-semibold">配置文件</h2>
					<button class="btn btn-sm btn-primary gap-1" @click="openAddFileDrawer">
						<Plus class="size-3.5" />
						添加文件
					</button>
				</div>
				<div v-if="fileListLoading" class="flex justify-center py-8">
					<span class="loading loading-spinner loading-md text-primary" />
				</div>
				<div
					v-else-if="files.length === 0"
					class="flex flex-col items-center gap-2 py-8 text-base-content/60"
				>
					<FileX class="size-10" />
					<span class="text-sm">暂无配置文件</span>
				</div>
				<div v-else class="overflow-x-auto">
					<table class="table">
						<thead>
							<tr class="text-base-content/60">
								<th>文件路径</th>
								<th>创建时间</th>
								<th>操作</th>
							</tr>
						</thead>
						<tbody>
							<tr v-for="file in files" :key="file.id" class="hover">
								<td>
									<code class="text-xs">{{ file.path }}</code>
								</td>
								<td class="cell-muted">{{ formatTime(file.created_at) }}</td>
								<td>
									<div class="flex items-center gap-3">
										<button class="link link-primary" @click="openFileDrawer(file.id)">查看</button>
										<button class="link link-primary" @click="openFileDrawer(file.id, true)">
											编辑
										</button>
										<button class="link link-error" @click="confirmDeleteFile(file.id)">
											删除
										</button>
									</div>
								</td>
							</tr>
						</tbody>
					</table>
				</div>
			</div>
		</div>

		<!-- Service config card -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-5">
				<div class="mb-4">
					<h2 class="font-semibold">服务镜像</h2>
					<p class="mt-1 text-xs text-base-content/60">
						读取 compose service 的当前 image，可通过弹窗修改。后续再扩展环境变量和目录挂载。
					</p>
				</div>
				<div v-if="serviceConfigListLoading" class="flex justify-center py-8">
					<span class="loading loading-spinner loading-md text-primary" />
				</div>
				<div
					v-else-if="serviceConfigError"
					class="rounded-box border border-base-200 bg-base-200/40 px-4 py-5 text-sm text-base-content/70"
				>
					{{ serviceConfigError }}
				</div>
				<div
					v-else-if="serviceConfigs.length === 0"
					class="flex flex-col items-center gap-2 py-8 text-base-content/60"
				>
					<span class="text-sm">暂无 compose service</span>
				</div>
				<div v-else class="overflow-x-auto">
					<table class="table">
						<thead>
							<tr class="text-base-content/60">
								<th>Service</th>
								<th>镜像</th>
								<th>已覆盖</th>
								<th>更新时间</th>
								<th>操作</th>
							</tr>
						</thead>
						<tbody>
							<tr v-for="serviceConfig in serviceConfigs" :key="serviceConfig.service_name" class="hover">
								<td>
									<code class="text-xs">{{ serviceConfig.service_name }}</code>
								</td>
								<td class="text-sm">
									<code
										v-if="getServiceDisplayImage(serviceConfig)"
										class="text-xs break-all"
									>
										{{ getServiceDisplayImage(serviceConfig) }}
									</code>
									<span v-else class="text-base-content/40">未配置 image</span>
								</td>
								<td class="text-sm">
									<span
										v-if="isServiceOverridden(serviceConfig)"
										class="badge badge-sm badge-primary badge-outline"
									>
										已覆盖
									</span>
									<span v-else class="text-base-content/40">—</span>
								</td>
								<td class="text-sm text-base-content/60">
									{{ serviceConfig.updated_at ? formatTime(serviceConfig.updated_at) : '—' }}
								</td>
								<td>
									<div class="flex items-center gap-3">
										<button class="link link-primary" @click="openEditServiceConfigModal(serviceConfig)">
											编辑
										</button>
										<button
											class="link link-error disabled:no-underline disabled:opacity-30"
											:disabled="serviceConfigSaving || !canResetServiceConfig(serviceConfig)"
											@click="confirmResetServiceConfig(serviceConfig)"
										>
											重置
										</button>
									</div>
								</td>
							</tr>
						</tbody>
					</table>
				</div>
			</div>
		</div>

		<!-- Route management card -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-5">
				<div class="flex items-center justify-between mb-4">
					<h2 class="font-semibold">路由托管</h2>
					<div class="tooltip tooltip-left" :data-tip="!application?.route_managed ? '请先在基本信息中启用路由托管' : undefined">
						<button
							class="btn btn-sm btn-primary gap-1"
							:disabled="!application?.route_managed"
							@click="openAddRouteModal"
						>
							<Plus class="size-3.5" />
							添加路由
						</button>
					</div>
				</div>
				<div v-if="routeListLoading" class="flex justify-center py-8">
					<span class="loading loading-spinner loading-md text-primary" />
				</div>
				<div
					v-else-if="appRoutes.length === 0"
					class="flex flex-col items-center gap-2 py-8 text-base-content/60"
				>
					<Network class="size-10" />
					<span class="text-sm">暂无路由配置</span>
				</div>
				<div v-else class="overflow-x-auto">
					<table class="table">
						<thead>
							<tr class="text-base-content/60">
								<th>Service</th>
								<th>域名</th>
								<th>端口</th>
								<th>操作</th>
							</tr>
						</thead>
						<tbody>
							<tr v-for="r in appRoutes" :key="r.id" class="hover">
								<td><code class="text-xs">{{ r.service_name }}</code></td>
								<td class="text-sm">{{ r.domain }}</td>
								<td class="text-sm">{{ r.port }}</td>
								<td>
									<div class="flex items-center gap-3">
										<button
											class="link link-primary"
											:class="{ 'opacity-30 pointer-events-none': !application?.route_managed }"
											@click="openEditRouteModal(r)"
										>编辑</button>
										<button class="link link-error" @click="confirmDeleteRoute(r.id)">删除</button>
									</div>
								</td>
							</tr>
						</tbody>
					</table>
				</div>
			</div>
		</div>

		<!-- File panel -->
		<Teleport to="body">
			<Transition
				enter-active-class="transition-transform duration-300 ease-out"
				enter-from-class="translate-x-full"
				enter-to-class="translate-x-0"
				leave-active-class="transition-transform duration-300 ease-in"
				leave-from-class="translate-x-0"
				leave-to-class="translate-x-full"
			>
				<div
					v-if="fileDrawerVisible"
					class="fixed inset-y-0 right-0 z-50 w-[720px] max-w-full bg-base-100 shadow-2xl flex flex-col border-l border-base-200"
				>
					<div class="flex items-center justify-between px-5 py-4 border-b border-base-200">
						<h3 class="font-semibold">
							{{
								currentFileId
									? (isEditingInDrawer ? '编辑文件' : '查看文件') + ': ' + currentFilePath
									: '新建文件'
							}}
						</h3>
						<button class="btn btn-sm btn-ghost btn-circle" @click="handleDrawerClose">
							<X class="size-4" />
						</button>
					</div>
					<div class="flex-1 overflow-auto p-5 flex flex-col gap-4">
						<div v-if="isEditingInDrawer || !currentFileId">
							<fieldset class="fieldset">
								<legend class="fieldset-legend">文件路径</legend>
								<input
									v-model="currentFilePath"
									type="text"
									class="input w-full"
									placeholder="例如: nginx.conf"
								/>
							</fieldset>
						</div>
						<div style="height: calc(100vh - 180px)" class="bg-[#1a202c] rounded-lg p-2">
							<CodeEditor
								v-if="!fileContentLoading"
								v-model:value="currentFileContent"
								:style="{ height: '100%' }"
								theme="vs-dark"
								:language="currentFileLanguage"
								:options="{
									readOnly: !isEditingInDrawer && !!currentFileId,
									minimap: { enabled: false },
									fontSize: 14,
									automaticLayout: true,
								}"
							/>
							<div v-else class="flex justify-center items-center h-full">
								<span class="loading loading-spinner loading-md text-primary" />
							</div>
						</div>
					</div>
					<div class="flex items-center justify-end gap-2 px-5 py-4 border-t border-base-200">
						<template v-if="!isEditingInDrawer && currentFileId">
							<button class="btn btn-sm btn-primary" @click="isEditingInDrawer = true">编辑</button>
							<button class="btn btn-sm btn-ghost" @click="handleDrawerClose">关闭</button>
						</template>
						<template v-else>
							<button
								class="btn btn-sm btn-primary"
								:disabled="fileContentLoading"
								@click="saveCurrentFile"
							>
								<span v-if="fileContentLoading" class="loading loading-spinner loading-xs" />
								保存
							</button>
							<button class="btn btn-sm btn-ghost" @click="handleDrawerClose">取消</button>
						</template>
					</div>
				</div>
			</Transition>
			<!-- Overlay -->
			<Transition
				enter-active-class="transition-opacity duration-300"
				enter-from-class="opacity-0"
				enter-to-class="opacity-100"
				leave-active-class="transition-opacity duration-300"
				leave-from-class="opacity-100"
				leave-to-class="opacity-0"
			>
				<div
					v-if="fileDrawerVisible"
					class="fixed inset-0 z-40 bg-black/30"
					@click="handleDrawerClose"
				/>
			</Transition>
		</Teleport>

		<!-- Edit basic info modal -->
		<dialog ref="editModalRef" class="modal">
			<div class="modal-box w-full max-w-lg">
				<h3 class="font-bold text-lg mb-4">编辑基本信息</h3>
				<div class="flex flex-col gap-3">
					<fieldset class="fieldset">
						<legend class="fieldset-legend">应用名称</legend>
						<input
							v-model="editForm.name"
							type="text"
							class="input w-full"
							:class="{ 'input-error': editErrors.name }"
						/>
						<p v-if="editErrors.name" class="fieldset-label text-error">{{ editErrors.name }}</p>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">应用编码</legend>
						<input :value="editForm.code" type="text" class="input w-full opacity-60" disabled />
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">镜像拉取策略</legend>
						<select v-model="editForm.image_pull_policy" class="select w-full">
							<option value="always">always</option>
							<option value="missing">missing</option>
							<option value="never">never</option>
						</select>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">路由托管</legend>
						<label class="flex items-center gap-3 cursor-pointer">
							<input v-model="editForm.route_managed" type="checkbox" class="toggle toggle-sm" />
							<span class="text-sm text-base-content/70">启用后，部署时将自动生成 Traefik 路由配置</span>
						</label>
					</fieldset>
				</div>
				<div class="modal-action">
					<button class="btn btn-primary" :disabled="operating" @click="handleEditOk">
						<span v-if="operating" class="loading loading-spinner loading-xs" />
						保存
					</button>
					<button class="btn btn-ghost" @click="editModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>

		<!-- Delete modal -->
		<dialog ref="deleteModalRef" class="modal">
			<div class="modal-box">
				<h3 class="font-bold text-lg">删除应用</h3>
				<p class="py-4 text-sm">
					确定要删除应用「
					<strong>{{ application?.name }}</strong>
					」吗？此操作不可撤销。
				</p>
				<label class="flex items-center gap-2 cursor-pointer mb-2">
					<input v-model="deleteDir" type="checkbox" class="checkbox checkbox-sm checkbox-error" />
					<span class="text-sm">同时删除应用工作目录（data/apps/{{ application?.code }}）</span>
				</label>
				<div class="modal-action">
					<button class="btn btn-error" :disabled="operating" @click="handleDeleteOk">
						<span v-if="operating" class="loading loading-spinner loading-xs" />
						删除
					</button>
					<button class="btn btn-ghost" @click="deleteModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>

		<!-- Delete file confirm modal -->
		<dialog ref="deleteFileModalRef" class="modal">
			<div class="modal-box">
				<h3 class="font-bold text-lg">删除文件</h3>
				<p class="py-4 text-sm">确定删除此配置文件？</p>
				<div class="modal-action">
					<button class="btn btn-error" :disabled="fileListLoading" @click="executeDeleteFile">
						<span v-if="fileListLoading" class="loading loading-spinner loading-xs" />
						删除
					</button>
					<button class="btn btn-ghost" @click="deleteFileModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>

		<!-- Edit service image modal -->
		<dialog ref="serviceConfigModalRef" class="modal">
			<div class="modal-box w-full max-w-md">
				<h3 class="font-bold text-lg mb-4">编辑服务镜像</h3>
				<div class="flex flex-col gap-3">
					<fieldset class="fieldset">
						<legend class="fieldset-legend">Service</legend>
						<input
							:value="activeServiceConfig?.service_name || ''"
							type="text"
							class="input w-full opacity-60"
							disabled
						/>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">镜像 Image</legend>
						<input
							v-model="serviceConfigForm.image"
							type="text"
							class="input w-full"
							placeholder="例如: nginx:1.28"
						/>
						<p class="fieldset-label">留空会移除单独覆盖，回退到 compose 中的 image。</p>
					</fieldset>
				</div>
				<div class="modal-action">
					<button
						class="btn btn-primary"
						:disabled="serviceConfigSaving || !activeServiceConfig || !serviceConfigDirty"
						@click="saveServiceConfig"
					>
						<span v-if="serviceConfigSaving" class="loading loading-spinner loading-xs" />
						确定
					</button>
					<button class="btn btn-ghost" @click="serviceConfigModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>

		<!-- Reset service image confirm modal -->
		<dialog ref="deleteServiceConfigModalRef" class="modal">
			<div class="modal-box">
				<h3 class="font-bold text-lg">重置服务镜像</h3>
				<p class="py-4 text-sm">
					确定删除
					<code class="text-xs">{{ pendingDeleteServiceName || '当前 service' }}</code>
					的镜像覆盖并回退到 compose 原值？
				</p>
				<div class="modal-action">
					<button class="btn btn-error" :disabled="serviceConfigSaving" @click="executeResetServiceConfig">
						<span v-if="serviceConfigSaving" class="loading loading-spinner loading-xs" />
						删除
					</button>
					<button class="btn btn-ghost" @click="cancelResetServiceConfig">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button @click="cancelResetServiceConfig">close</button></form>
		</dialog>

		<!-- Add/Edit route modal -->
		<dialog ref="routeModalRef" class="modal">
			<div class="modal-box w-full max-w-md">
				<h3 class="font-bold text-lg mb-4">{{ editingRouteId ? '编辑路由' : '添加路由' }}</h3>
				<div class="flex flex-col gap-3">
					<fieldset class="fieldset">
						<legend class="fieldset-legend">Service</legend>
						<div class="dropdown w-full">
							<label
								ref="serviceDropdownRef"
								tabindex="0"
								class="input w-full flex items-center justify-between cursor-pointer"
								:class="{ 'input-error': routeFormErrors.service_name }"
							>
								<span :class="routeForm.service_name ? '' : 'text-base-content/40'">
									{{ routeForm.service_name || '请选择 service' }}
								</span>
								<ChevronDown class="size-4 text-base-content/40 shrink-0" />
							</label>
							<ul
								tabindex="0"
								class="dropdown-content menu bg-base-100 rounded-box border border-base-200 shadow-lg z-50 w-full flex-nowrap p-0 mt-1"
							>
								<li v-if="composeServices.length === 0">
									<span class="text-xs text-base-content/50 px-3 py-2">暂无 service</span>
								</li>
								<li v-for="s in composeServices" :key="s.service_name">
									<a
										class="text-xs px-3 py-1.5 rounded-none block truncate"
										:class="{ 'bg-primary/10 font-medium': s.service_name === routeForm.service_name }"
										@mousedown.prevent="selectComposeService(s)"
									>
										{{ s.service_name }}
										<span class="text-base-content/40 ml-1">{{ s.default_domain }}:{{ s.default_port }}</span>
									</a>
								</li>
							</ul>
						</div>
						<p v-if="routeFormErrors.service_name" class="fieldset-label text-error">
							{{ routeFormErrors.service_name }}
						</p>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">域名</legend>
						<input
							v-model="routeForm.domain"
							type="text"
							class="input w-full"
							placeholder="例如: myapp.lvh.me"
							:class="{ 'input-error': routeFormErrors.domain }"
						/>
						<p v-if="routeFormErrors.domain" class="fieldset-label text-error">
							{{ routeFormErrors.domain }}
						</p>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">容器端口</legend>
						<input
							v-model.number="routeForm.port"
							type="number"
							class="input w-full"
							placeholder="例如: 80"
							min="1"
							max="65535"
							:class="{ 'input-error': routeFormErrors.port }"
						/>
						<p v-if="routeFormErrors.port" class="fieldset-label text-error">
							{{ routeFormErrors.port }}
						</p>
					</fieldset>
				</div>
				<div class="modal-action">
					<button class="btn btn-primary" :disabled="routeLoading" @click="handleRouteOk">
						<span v-if="routeLoading" class="loading loading-spinner loading-xs" />
						保存
					</button>
					<button class="btn btn-ghost" @click="routeModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>

		<!-- Delete route confirm modal -->
		<dialog ref="deleteRouteModalRef" class="modal">
			<div class="modal-box">
				<h3 class="font-bold text-lg">删除路由</h3>
				<p class="py-4 text-sm">确定删除此路由配置？</p>
				<div class="modal-action">
					<button class="btn btn-error" :disabled="routeLoading" @click="executeDeleteRoute">
						<span v-if="routeLoading" class="loading loading-spinner loading-xs" />
						删除
					</button>
					<button class="btn btn-ghost" @click="deleteRouteModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>
	</div>
</template>

<script setup lang="ts">
import { ArrowLeft, ChevronDown, Download, FileX, Network, Plus, Rocket, X } from 'lucide-vue-next';
import { CodeEditor } from 'monaco-editor-vue3';
import { computed, onMounted, reactive, ref, watch } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { applicationApi } from '@/api/cd/application';
import { deploymentApi } from '@/api/cd/deployments';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type {
	Application,
	ApplicationRoute,
	ApplicationServiceConfig,
	ComposeServiceResp,
	ConfigFile,
} from '@/types/cd/application';
import { appStatusLabel } from '@/utils/status';
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

const application = ref<Application>();
const files = ref<ConfigFile[]>([]);
const appRoutes = ref<ApplicationRoute[]>([]);
const composeServices = ref<ComposeServiceResp[]>([]);
const serviceConfigs = ref<ApplicationServiceConfig[]>([]);
const selectedServiceName = ref('');
const serviceConfigError = ref('');

const editModalRef = ref<HTMLDialogElement>();
const deleteModalRef = ref<HTMLDialogElement>();
const deleteFileModalRef = ref<HTMLDialogElement>();
const serviceConfigModalRef = ref<HTMLDialogElement>();
const deleteServiceConfigModalRef = ref<HTMLDialogElement>();
const routeModalRef = ref<HTMLDialogElement>();
const deleteRouteModalRef = ref<HTMLDialogElement>();
const pendingDeleteFileId = ref('');
const pendingDeleteServiceName = ref('');
const pendingDeleteRouteId = ref('');
const editingRouteId = ref('');
const deleteDir = ref(false);

const routeForm = reactive({ service_name: '', domain: '', port: 80 });
const routeFormErrors = reactive({ service_name: '', domain: '', port: '' });
const serviceDropdownRef = ref<HTMLElement>();
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

const envs = computed(() =>
	files.value.filter((f) => f.path.match(/^\.env(\..+)?$/)).map((f) => f.path)
);
const activeServiceConfig = computed(() =>
	serviceConfigs.value.find((item) => item.service_name === selectedServiceName.value)
);
const currentServiceImage = computed(
	() => (activeServiceConfig.value ? getServiceDisplayImage(activeServiceConfig.value) : '')
);
const serviceConfigDirty = computed(
	() => serviceConfigForm.image.trim() !== currentServiceImage.value.trim()
);

const badgeMap: Record<string, string> = {
	deployed: 'badge-outline badge-success',
	deploy_failed: 'badge-outline badge-error',
	deploying: 'badge-outline badge-info',
	undeployed: 'badge-ghost',
};
function appBadgeClass(s: string) {
	return badgeMap[s] ?? 'badge-ghost';
}

watch(
	currentServiceImage,
	(value) => {
		serviceConfigForm.image = value;
	},
	{ immediate: true }
);

const currentFileLanguage = computed(() => {
	const p = currentFilePath.value.toLowerCase();
	if (p.endsWith('.sh') || p.endsWith('.bash')) {
		return 'shell';
	}
	if (p.startsWith('.env') || p.endsWith('.ini') || p.endsWith('.properties')) {
		return 'ini';
	}
	return 'yaml';
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
			const defaultEnv = envs.value.includes('.env') ? '.env' : undefined;
			const res = await applicationApi.deploy(applicationId, undefined, defaultEnv);
			toast.success(`部署已触发`);
			router.push(`/cd/deployments/${res.deployment_id}`);
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '触发部署失败');
	}
}

async function handleDeployWithEnv(env: string) {
	try {
		await executeOp(async () => {
			const res = await applicationApi.deploy(applicationId, undefined, env);
			toast.success(`部署已触发`);
			router.push(`/cd/deployments/${res.deployment_id}`);
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
			while (true) {
				await delayAsync(3000);
				try {
					const detail = await deploymentApi.get(res.deployment_id);
					if (['ran_to_completion', 'faulted', 'canceled'].includes(detail.status)) {
						if (application.value) {
							application.value.status =
								detail.status === 'ran_to_completion' ? 'undeployed' : 'deploy_failed';
						}
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
			router.push(`/cd/deployments/${res.deployment_id}`);
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
	editModalRef.value?.showModal();
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
			editModalRef.value?.close();
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
	deleteModalRef.value?.showModal();
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
	serviceConfigModalRef.value?.showModal();
}

function confirmResetServiceConfig(serviceConfig: ApplicationServiceConfig) {
	if (!canResetServiceConfig(serviceConfig)) {
		return;
	}
	pendingDeleteServiceName.value = serviceConfig.service_name;
	deleteServiceConfigModalRef.value?.showModal();
}

function cancelResetServiceConfig() {
	pendingDeleteServiceName.value = '';
	deleteServiceConfigModalRef.value?.close();
}

async function executeResetServiceConfig() {
	if (!pendingDeleteServiceName.value) {
		return;
	}
	try {
		await executeServiceConfigSave(async () => {
			const saved = await applicationApi.updateServiceConfig(applicationId, pendingDeleteServiceName.value, null);
			const idx = serviceConfigs.value.findIndex((item) => item.service_name === saved.service_name);
			if (idx >= 0) {
				serviceConfigs.value[idx] = saved;
			} else {
				serviceConfigs.value.push(saved);
			}
			if (selectedServiceName.value === saved.service_name) {
				serviceConfigForm.image = getServiceDisplayImage(saved);
				serviceConfigModalRef.value?.close();
			}
			deleteServiceConfigModalRef.value?.close();
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
			const saved = await applicationApi.updateServiceConfig(applicationId, active.service_name, serviceConfigForm.image);
			const idx = serviceConfigs.value.findIndex((item) => item.service_name === saved.service_name);
			if (idx >= 0) {
				serviceConfigs.value[idx] = saved;
			} else {
				serviceConfigs.value.push(saved);
			}
			selectedServiceName.value = saved.service_name;
			serviceConfigForm.image = getServiceDisplayImage(saved);
			serviceConfigModalRef.value?.close();
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
	deleteFileModalRef.value?.showModal();
}

async function executeDeleteFile() {
	try {
		await executeFileList(async () => {
			await applicationApi.deleteFile(applicationId, pendingDeleteFileId.value);
			toast.success('删除成功');
			deleteFileModalRef.value?.close();
			await loadFiles();
			await loadServiceConfigs();
		});
	} catch {
		toast.error('删除失败');
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
		toast.error(error instanceof Error ? error.message : '解析 docker-compose 失败，无法配置路由');
	}
}

function selectComposeService(s: ComposeServiceResp) {
	routeForm.service_name = s.service_name;
	routeForm.domain = s.default_domain;
	routeForm.port = s.default_port;
	serviceDropdownRef.value?.blur();
}

async function openAddRouteModal() {
	editingRouteId.value = '';
	Object.assign(routeForm, { service_name: '', domain: '', port: 80 });
	Object.assign(routeFormErrors, { service_name: '', domain: '', port: '' });
	await loadComposeServices();
	if (composeServices.value.length === 0) {
		return;
	}
	routeModalRef.value?.showModal();
	serviceDropdownRef.value?.blur();
}

async function openEditRouteModal(r: ApplicationRoute) {
	editingRouteId.value = r.id;
	Object.assign(routeForm, { service_name: r.service_name, domain: r.domain, port: r.port });
	Object.assign(routeFormErrors, { service_name: '', domain: '', port: '' });
	await loadComposeServices();
	if (composeServices.value.length === 0) {
		return;
	}
	routeModalRef.value?.showModal();
	serviceDropdownRef.value?.blur();
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
			routeModalRef.value?.close();
			await loadRoutes();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '保存失败');
	}
}

function confirmDeleteRoute(routeId: string) {
	pendingDeleteRouteId.value = routeId;
	deleteRouteModalRef.value?.showModal();
}

async function executeDeleteRoute() {
	try {
		await executeRoute(async () => {
			await applicationApi.deleteRoute(applicationId, pendingDeleteRouteId.value);
			toast.success('删除成功');
			deleteRouteModalRef.value?.close();
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
