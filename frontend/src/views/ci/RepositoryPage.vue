<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold">代码仓库</h1>
			<div class="flex items-center gap-2">
				<label class="input input-sm flex items-center gap-1 w-52">
					<Search class="size-3.5 text-base-content/40 shrink-0" />
					<input
						v-model="searchText"
						type="text"
						class="grow"
						placeholder="搜索名称/地址"
						@keydown.enter="handleSearch"
					/>
					<button
						v-if="searchText"
						class="text-base-content/40 hover:text-base-content/70"
						@click="
							searchText = '';
							handleSearch();
						"
					>
						<X class="size-3" />
					</button>
				</label>
				<button
					class="btn btn-sm btn-primary"
					:disabled="status === 'loading'"
					@click="handleSearch"
				>
					<span v-if="status === 'loading'" class="loading loading-spinner loading-xs" />
					搜索
				</button>
				<button class="btn btn-sm btn-primary gap-1.5" @click="openCreateModal">
					<Plus class="size-4" />
					新建仓库
				</button>
			</div>
		</div>

		<div class="card bg-base-100 shadow-sm overflow-x-auto">
			<table class="table min-h-48">
				<thead>
					<tr class="text-base-content/60">
						<th>名称</th>
						<th>编码</th>
						<th>地址</th>
						<th>Git 凭据</th>
						<th>创建时间</th>
						<th>操作</th>
					</tr>
				</thead>
				<tbody>
					<tr v-if="status === 'loading'">
						<td colspan="6" class="text-center py-8">
							<span class="loading loading-spinner loading-md text-primary" />
						</td>
					</tr>
					<tr v-else-if="status === 'error'">
						<td colspan="6" class="text-center py-8 text-error">{{ error }}</td>
					</tr>
					<tr v-else-if="repositories.length === 0">
						<td colspan="6" class="text-center py-8 text-base-content/60">暂无数据</td>
					</tr>
					<tr v-for="p in repositories" :key="p.id" class="hover">
						<td>
							<router-link :to="`/ci/repository/${p.id}`" class="link link-primary font-medium">
								{{ p.name }}
							</router-link>
						</td>
						<td class="text-base-content/70">{{ p.code }}</td>
						<td class="cell-muted max-w-xs truncate">{{ p.repository_url }}</td>
						<td class="cell-muted">
							<router-link
								v-if="p.git_credential_id"
								:to="`/ci/credential/${p.git_credential_id}`"
								class="link link-primary"
							>
								已配置
							</router-link>
							<span v-else>—</span>
						</td>
						<td class="cell-muted">{{ formatTime(p.created_at) }}</td>
						<td>
							<router-link :to="`/ci/repository/${p.id}`" class="link link-primary">
								查看
							</router-link>
						</td>
					</tr>
				</tbody>
			</table>
			<div v-if="totalPages > 0" class="flex justify-end p-3 border-t border-base-200">
				<div class="join">
					<button
						v-for="p in totalPages"
						:key="p"
						class="join-item btn btn-sm"
						:class="p === pagination.current ? 'btn-primary' : 'btn-ghost'"
						@click="goPage(p)"
					>
						{{ p }}
					</button>
				</div>
			</div>
		</div>

		<!-- Create modal -->
		<dialog ref="createModalRef" class="modal">
			<div class="modal-box w-full max-w-lg">
				<h3 class="font-bold text-lg mb-4">新建仓库</h3>
				<div v-if="modalStatus === 'loading'" class="flex justify-center py-8">
					<span class="loading loading-spinner loading-md text-primary" />
				</div>
				<div v-else class="flex flex-col gap-3">
					<fieldset class="fieldset">
						<legend class="fieldset-legend">名称</legend>
						<input
							v-model="form.name"
							type="text"
							class="input w-full"
							:class="{ 'input-error': errors.name }"
							placeholder="例如: my-backend"
						/>
						<p v-if="errors.name" class="fieldset-label text-error">{{ errors.name }}</p>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">仓库编码</legend>
						<input
							v-model="form.code"
							type="text"
							class="input w-full"
							:class="{ 'input-error': errors.code }"
							placeholder="例如: my-backend（固化工作目录，创建后不可修改）"
						/>
						<p v-if="errors.code" class="fieldset-label text-error">{{ errors.code }}</p>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">仓库地址</legend>
						<input
							v-model="form.repository_url"
							type="text"
							class="input w-full"
							:class="{ 'input-error': errors.repository_url }"
							placeholder="git@github.com:user/repo.git"
						/>
						<p v-if="errors.repository_url" class="fieldset-label text-error">
							{{ errors.repository_url }}
						</p>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">Git 凭据（可选）</legend>
						<select v-model="form.git_credential_id" class="select w-full">
							<option value="">不使用凭据</option>
							<option v-for="cred in gitCredentials" :key="cred.id" :value="cred.id">
								{{ cred.name }} ({{ credentialTypeLabels[cred.type] }})
							</option>
						</select>
					</fieldset>
				</div>
				<div class="modal-action">
					<button
						class="btn btn-primary"
						:disabled="operating || modalStatus === 'loading'"
						@click="handleCreateOk"
					>
						<span v-if="operating" class="loading loading-spinner loading-xs" />
						创建
					</button>
					<button class="btn btn-ghost" @click="createModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>
	</div>
</template>

<script setup lang="ts">
import { Plus, Search, X } from 'lucide-vue-next';
import { computed, onMounted, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { credentialApi, repositoryApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type { Credential } from '@/types/ci/credential';
import type { RepositoryListItem } from '@/types/ci/repository';
import { credentialTypeLabels } from '@/types/ci/credential';
import { formatTime } from '@/utils/time';

const router = useRouter();
const toast = useToast();
const { status, error, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();
const { status: modalStatus, execute: executeModal } = useStatusAsync();

const repositories = ref<RepositoryListItem[]>([]);
const credentials = ref<Credential[]>([]);
const gitCredentials = computed(() =>
	credentials.value.filter((c) => c.type === 'git_ssh' || c.type === 'git_token' || c.type === 'gitee_token')
);

const pagination = reactive({ current: 1, pageSize: 20, total: 0 });
const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));
const searchText = ref('');

const createModalRef = ref<HTMLDialogElement>();

const form = reactive({
	name: '',
	code: '',
	repository_url: '',
	git_credential_id: '',
});
const errors = reactive({
	name: '',
	code: '',
	repository_url: '',
});

function validate() {
	errors.name = form.name.trim() ? '' : '请输入名称';
	errors.code = /^[a-z0-9-]+$/.test(form.code.trim()) ? '' : '编码只能包含小写字母、数字和连字符';
	errors.repository_url = form.repository_url.trim() ? '' : '请输入仓库地址';
	return !errors.name && !errors.code && !errors.repository_url;
}

async function fetchProjects() {
	try {
		await execute(async () => {
			const res = await repositoryApi.list({
				page: pagination.current,
				per_page: pagination.pageSize,
				search: searchText.value || undefined,
			});
			repositories.value = res.items;
			pagination.total = res.total;
		});
	} catch {
		toast.error('获取项目列表失败');
	}
}

function handleSearch() {
	pagination.current = 1;
	fetchProjects();
}

function goPage(p: number) {
	pagination.current = p;
	fetchProjects();
}

async function openCreateModal() {
	Object.assign(form, {
		name: '',
		code: '',
		repository_url: '',
		git_credential_id: '',
	});
	Object.assign(errors, { name: '', code: '', repository_url: '' });
	createModalRef.value?.showModal();
	try {
		await executeModal(async () => {
			const credRes = await credentialApi.list({ per_page: 100 });
			credentials.value = credRes.items;
		});
	} catch {
		toast.error('加载表单数据失败');
	}
}

async function handleCreateOk() {
	if (!validate()) {
		return;
	}
	try {
		await executeOp(async () => {
			const repository = await repositoryApi.create({
				name: form.name,
				code: form.code,
				repository_url: form.repository_url,
				git_credential_id: form.git_credential_id || undefined,
			});
			toast.success('创建成功');
			createModalRef.value?.close();
			router.push(`/ci/repository/${repository.id}`);
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '创建失败');
	}
}

onMounted(fetchProjects);
</script>
