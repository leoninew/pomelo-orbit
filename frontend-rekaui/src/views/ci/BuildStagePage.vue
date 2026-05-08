<script setup lang="ts">
import { Plus } from 'lucide-vue-next';
import { computed, onMounted, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import {
	DialogClose,
	DialogContent,
	DialogDescription,
	DialogOverlay,
	DialogPortal,
	DialogRoot,
	DialogTitle,
	ToolbarRoot,
} from 'reka-ui';
import { buildStageApi } from '@/api/ci';
import ListPagination from '@/components/ListPagination.vue';
import SearchControl from '@/components/SearchControl.vue';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type { BuildStage } from '@/types/ci/template';
import { formatTime } from '@/utils/time';

const router = useRouter();
const toast = useToast();
const { status, error, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();
const { loading: duplicating, execute: executeDuplicate } = useStatusAsync();

const stages = ref<BuildStage[]>([]);
const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));
const searchText = ref('');
const isModalOpen = ref(false);

const form = reactive({ name: '', image: '', script: '', description: '' });
const errors = reactive({ name: '', image: '', script: '' });

function validate() {
	errors.name = form.name.trim() ? '' : '请输入名称';
	errors.image = form.image.trim() ? '' : '请输入镜像';
	errors.script = form.script.trim() ? '' : '请输入脚本';
	return !errors.name && !errors.image && !errors.script;
}

async function fetchStages() {
	try {
		await execute(async () => {
			const res = await buildStageApi.list({
				page: pagination.current,
				per_page: pagination.pageSize,
				search: searchText.value || undefined,
			});
			stages.value = res.items;
			pagination.total = res.total;
		});
	} catch {
		toast.error('获取 Stage 列表失败');
	}
}

function handleSearch() {
	if (status.value === 'loading') {
		return;
	}
	pagination.current = 1;
	fetchStages();
}

function goPage(p: number) {
	pagination.current = p;
	fetchStages();
}

function handlePageSizeChange(pageSize: number) {
	pagination.pageSize = pageSize;
	pagination.current = 1;
	fetchStages();
}

function openCreateModal() {
	Object.assign(form, { name: '', image: '', script: '', description: '' });
	Object.assign(errors, { name: '', image: '', script: '' });
	isModalOpen.value = true;
}

async function handleModalOk() {
	if (!validate()) {
		return;
	}
	try {
		await executeOp(async () => {
			await buildStageApi.create({
				name: form.name,
				image: form.image,
				script: form.script,
				description: form.description,
			});
			toast.success('创建成功');
			isModalOpen.value = false;
			fetchStages();
		});
	} catch (err) {
		toast.error(err instanceof Error ? err.message : '操作失败');
	}
}

async function handleDuplicate(id: string) {
	try {
		await executeDuplicate(async () => {
			const newStage = await buildStageApi.duplicate(id);
			toast.success('复制成功');
			router.push(`/ci/build-stage/${newStage.id}`);
		});
	} catch (err) {
		toast.error(err instanceof Error ? err.message : '复制失败');
	}
}

onMounted(fetchStages);
</script>

<template>
	<div class="space-y-6">
		<ToolbarRoot class="flex items-center justify-between gap-6" aria-label="构建阶段工具栏">
			<SearchControl
				v-model="searchText"
				placeholder="搜索名称/描述"
				:loading="status === 'loading'"
				@search="handleSearch"
			/>
			<button
				class="flex h-10 items-center gap-2 rounded-md bg-primary px-5 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
				@click="openCreateModal"
			>
				<Plus class="size-4" />
				新建构建
			</button>
		</ToolbarRoot>

		<div class="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
			<div v-if="status === 'loading'" class="flex justify-center py-16">
				<div class="size-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary" />
			</div>
			<div v-else-if="status === 'error'" class="text-center py-16 text-destructive">
				<p class="text-sm">{{ error }}</p>
			</div>
			<div v-else-if="stages.length === 0" class="text-center py-16 text-muted-foreground">
				<p class="text-sm">暂无数据</p>
			</div>
			<div v-else class="overflow-x-auto">
				<table class="app-table-list min-w-[1120px]">
					<thead>
						<tr>
							<th>名称</th>
							<th>镜像</th>
							<th>版本</th>
							<th>制品</th>
							<th>描述</th>
							<th>创建时间</th>
							<th>操作</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="s in stages" :key="s.id">
							<td>
								<router-link :to="`/ci/build-stage/${s.id}`" class="text-primary hover:underline">
									{{ s.name }}
								</router-link>
							</td>
							<td class="max-w-64 truncate text-foreground" :title="s.image">{{ s.image }}</td>
							<td>
								<span class="inline-block rounded bg-muted px-2 py-0.5 text-sm text-muted-foreground">v{{ s.version }}</span>
							</td>
							<td class="text-foreground">{{ s.artifacts?.length ?? 0 }}</td>
							<td class="max-w-xs truncate text-muted-foreground" :title="s.description || undefined">
								{{ s.description || '—' }}
							</td>
							<td class="text-muted-foreground">{{ formatTime(s.created_at) }}</td>
							<td>
								<router-link :to="`/ci/build-stage/${s.id}`" class="text-primary hover:underline">
									查看
								</router-link>
								<button
									class="ml-3 text-primary hover:underline disabled:cursor-not-allowed disabled:opacity-50"
									:disabled="duplicating"
									@click="handleDuplicate(s.id)"
								>
									复制
								</button>
							</td>
						</tr>
					</tbody>
				</table>
			</div>
		</div>

		<ListPagination
			:current="pagination.current"
			:page-size="pagination.pageSize"
			:total="pagination.total"
			:total-pages="totalPages"
			@change-page="goPage"
			@change-page-size="handlePageSizeChange"
		/>

		<!-- Create modal -->
		<DialogRoot v-model:open="isModalOpen">
			<DialogPortal>
				<DialogOverlay class="fixed inset-0 z-50 bg-black/50 data-[state=open]:animate-overlayShow" />
				<DialogContent
					class="fixed left-1/2 top-1/2 z-50 max-h-[90vh] w-[min(672px,calc(100vw-32px))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-lg border border-border bg-card p-6 shadow-xl outline-none data-[state=open]:animate-contentShow"
				>
					<div class="mb-5 space-y-1">
						<DialogTitle class="text-lg font-semibold text-foreground">新建构建</DialogTitle>
						<DialogDescription class="text-sm text-muted-foreground">
							创建一个可被流水线模板复用的构建阶段。
						</DialogDescription>
					</div>
					
					<div class="space-y-4">
						<div class="grid grid-cols-2 gap-3">
							<div class="space-y-1.5">
								<label class="block text-sm font-medium text-foreground">名称</label>
								<input
									v-model="form.name"
									type="text"
									class="w-full rounded-md border bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
									:class="errors.name ? 'border-destructive' : 'border-input'"
									placeholder="例如: build"
								/>
								<p v-if="errors.name" class="text-xs text-destructive">{{ errors.name }}</p>
							</div>
							<div class="space-y-1.5">
								<label class="block text-sm font-medium text-foreground">镜像</label>
								<input
									v-model="form.image"
									type="text"
									class="w-full rounded-md border bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
									:class="errors.image ? 'border-destructive' : 'border-input'"
									placeholder="例如: alpine:latest"
								/>
								<p v-if="errors.image" class="text-xs text-destructive">{{ errors.image }}</p>
							</div>
						</div>
						<div class="space-y-1.5">
							<label class="block text-sm font-medium text-foreground">脚本</label>
							<textarea
								v-model="form.script"
								class="w-full rounded-md border bg-background px-3 py-2 font-mono text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
								:class="errors.script ? 'border-destructive' : 'border-input'"
								rows="6"
								placeholder="echo hello&#10;echo world"
							/>
							<p v-if="errors.script" class="text-xs text-destructive">{{ errors.script }}</p>
						</div>
						<div class="space-y-1.5">
							<label class="block text-sm font-medium text-foreground">描述（可选）</label>
							<input
								v-model="form.description"
								type="text"
								class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
								placeholder="简短描述"
							/>
						</div>
					</div>
					
					<div class="mt-6 flex justify-end gap-2">
						<DialogClose as-child>
							<button class="rounded-md border border-input bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted/50">
								取消
							</button>
						</DialogClose>
						<button
							class="flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50"
							:disabled="operating"
							@click="handleModalOk"
						>
							<span v-if="operating" class="size-4 animate-spin rounded-full border-2 border-primary-foreground border-t-transparent" />
							保存
						</button>
					</div>
				</DialogContent>
			</DialogPortal>
		</DialogRoot>
	</div>
</template>
