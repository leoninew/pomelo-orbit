<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold flex items-center gap-2">
				Pipeline Run
				<span v-if="run" class="badge badge-sm" :class="runBadgeClass(run.status)">{{ run.status }}</span>
			</h1>
			<div class="flex items-center gap-2">
				<button class="btn btn-sm btn-ghost gap-1" @click="$router.push('/ci/runs')"><ArrowLeft class="size-4" />返回</button>
				<button v-if="run?.status === 'failed' || run?.status === 'success'" class="btn btn-sm btn-primary" @click="handleRetry">重试</button>
				<button v-if="run?.status === 'waiting' || run?.status === 'running'" class="btn btn-sm btn-error btn-ghost" @click="cancelModalRef?.showModal()">取消</button>
			</div>
		</div>

		<!-- Basic info -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-5">
				<h2 class="font-semibold mb-3">基本信息</h2>
				<div v-if="loading" class="flex justify-center py-6"><span class="loading loading-spinner loading-md text-primary" /></div>
				<dl v-else-if="run" class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-3 text-sm">
					<div class="flex gap-2"><dt class="text-base-content/50 w-24 shrink-0">Run ID</dt><dd class="font-mono text-xs">{{ run.id }}</dd></div>
					<div class="flex gap-2"><dt class="text-base-content/50 w-24 shrink-0">Project</dt><dd><router-link :to="`/ci/projects/${run.project_id}`" class="link link-primary text-xs">{{ run.project_id }}</router-link></dd></div>
					<div class="flex gap-2"><dt class="text-base-content/50 w-24 shrink-0">触发方式</dt><dd><span class="badge badge-xs badge-ghost">{{ run.trigger }}</span></dd></div>
					<div class="flex gap-2"><dt class="text-base-content/50 w-24 shrink-0">Ref</dt><dd class="text-base-content/60">{{ run.trigger_ref }}</dd></div>
					<div class="flex gap-2"><dt class="text-base-content/50 w-24 shrink-0">重试自</dt><dd><router-link v-if="run.retry_of" :to="`/ci/runs/${run.retry_of}`" class="link link-primary text-xs">{{ run.retry_of }}</router-link><span v-else class="text-base-content/40">—</span></dd></div>
					<div class="flex gap-2"><dt class="text-base-content/50 w-24 shrink-0">开始时间</dt><dd class="text-xs text-base-content/60">{{ run.started_at ? formatTime(run.started_at) : '—' }}</dd></div>
					<div class="flex gap-2"><dt class="text-base-content/50 w-24 shrink-0">结束时间</dt><dd class="text-xs text-base-content/60">{{ run.finished_at ? formatTime(run.finished_at) : '—' }}</dd></div>
					<div class="flex gap-2"><dt class="text-base-content/50 w-24 shrink-0">创建时间</dt><dd class="text-xs text-base-content/60">{{ formatTime(run.created_at) }}</dd></div>
				</dl>
			</div>
		</div>

		<!-- Jobs -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-5">
				<h2 class="font-semibold mb-3">Jobs</h2>
				<div v-if="jobsLoading" class="flex justify-center py-6"><span class="loading loading-spinner loading-md text-primary" /></div>
				<div v-else-if="jobs.length === 0" class="text-sm text-base-content/40 py-4 text-center">暂无 Job 记录</div>
				<table v-else class="table">
					<thead><tr class="text-base-content/60"><th>Job 名称</th><th>状态</th><th>开始时间</th><th>结束时间</th></tr></thead>
					<tbody>
						<tr v-for="j in jobs" :key="j.id" class="hover cursor-pointer" @click="showJobLogs(j)">
							<td class="link link-primary">{{ j.name }}</td>
							<td><span class="badge badge-sm" :class="jobBadgeClass(j.status)">{{ j.status }}</span></td>
							<td class="cell-muted">{{ j.started_at ? formatTime(j.started_at) : '—' }}</td>
							<td class="cell-muted">{{ j.finished_at ? formatTime(j.finished_at) : '—' }}</td>
						</tr>
					</tbody>
				</table>
			</div>
		</div>

		<!-- Artifacts -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-5">
				<h2 class="font-semibold mb-3">制品</h2>
				<div v-if="artifactsLoading" class="flex justify-center py-6"><span class="loading loading-spinner loading-md text-primary" /></div>
				<div v-else-if="artifacts.length === 0" class="text-sm text-base-content/40 py-4 text-center">暂无制品</div>
				<table v-else class="table">
					<thead><tr class="text-base-content/60"><th>Job</th><th>类型</th><th>名称</th><th>路径</th><th>创建时间</th></tr></thead>
					<tbody>
						<tr v-for="a in artifacts" :key="a.id" class="hover">
							<td>{{ a.job_name }}</td>
							<td><span class="badge badge-xs badge-ghost">{{ a.type }}</span></td>
							<td>{{ a.name }}</td>
							<td class="cell-muted max-w-xs truncate">{{ a.path }}</td>
							<td class="cell-muted">{{ formatTime(a.created_at) }}</td>
						</tr>
					</tbody>
				</table>
			</div>
		</div>

		<!-- Job logs drawer -->
		<div class="drawer drawer-end" :class="{ 'drawer-open': showLogsDrawer }">
			<input id="job-logs-drawer" type="checkbox" class="drawer-toggle" :checked="showLogsDrawer" @change="showLogsDrawer = ($event.target as HTMLInputElement).checked" />
			<div class="drawer-side z-40">
				<label for="job-logs-drawer" class="drawer-overlay" @click="showLogsDrawer = false" />
				<div class="w-[800px] max-w-full bg-base-100 h-full flex flex-col">
					<div class="flex items-center justify-between px-5 py-4 border-b border-base-200">
						<h3 class="font-semibold">Job: {{ currentJob?.name }}</h3>
						<button class="btn btn-sm btn-ghost btn-circle" @click="showLogsDrawer = false"><X class="size-4" /></button>
					</div>
					<div class="flex-1 overflow-auto p-5 flex flex-col gap-4">
						<dl v-if="currentJob" class="grid grid-cols-1 gap-y-2 text-sm">
							<div class="flex gap-2"><dt class="text-base-content/50 w-20 shrink-0">状态</dt><dd><span class="badge badge-sm" :class="jobBadgeClass(currentJob.status)">{{ currentJob.status }}</span></dd></div>
							<div v-if="currentJob.error_message" class="flex gap-2"><dt class="text-base-content/50 w-20 shrink-0">错误</dt><dd class="text-error text-xs">{{ currentJob.error_message }}</dd></div>
						</dl>
						<div v-if="logsLoading" class="flex justify-center py-8"><span class="loading loading-spinner loading-md text-primary" /></div>
						<div v-else class="flex-1 bg-neutral rounded-box p-4 overflow-auto min-h-64">
							<pre v-if="logsText" class="text-neutral-content font-mono text-xs leading-relaxed whitespace-pre-wrap break-all">{{ logsText }}</pre>
							<div v-else class="flex flex-col items-center gap-2 py-8 text-neutral-content/40"><FileX class="size-8" /><span class="text-sm">暂无日志</span></div>
						</div>
					</div>
				</div>
			</div>
		</div>

		<!-- Cancel confirm modal -->
		<dialog ref="cancelModalRef" class="modal">
			<div class="modal-box">
				<h3 class="font-bold text-lg">取消 Run</h3>
				<p class="py-4">确定取消此 Run？</p>
				<div class="modal-action">
					<button class="btn btn-error" @click="handleCancel">确定</button>
					<button class="btn btn-ghost" @click="cancelModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>
	</div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { ArrowLeft, X, FileX } from 'lucide-vue-next';
import { jobApi, pipelineRunApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import { formatTime } from '@/utils/time';
import type { Artifact, Job, JobLog, PipelineRun } from '@/types/api';

const route = useRoute();
const router = useRouter();
const runId = route.params.id as string;
const toast = useToast();

const { loading, execute } = useStatusAsync();
const { loading: jobsLoading, execute: executeJobs } = useStatusAsync();
const { loading: artifactsLoading, execute: executeArtifacts } = useStatusAsync();
const { loading: logsLoading, execute: executeLogs } = useStatusAsync();

const run = ref<PipelineRun>();
const jobs = ref<Job[]>([]);
const artifacts = ref<Artifact[]>([]);
const logs = ref<JobLog | null>(null);
const currentJob = ref<Job>();
const showLogsDrawer = ref(false);
const cancelModalRef = ref<HTMLDialogElement>();

const logsText = computed(() => logs.value?.content ?? '');

const runBadgeMap: Record<string, string> = { success: 'badge-success', failed: 'badge-error', running: 'badge-info', waiting: 'badge-warning', canceled: 'badge-ghost' };
const jobBadgeMap: Record<string, string> = { success: 'badge-success', failed: 'badge-error', running: 'badge-info', pending: 'badge-warning', skipped: 'badge-ghost' };
function runBadgeClass(s: string) { return runBadgeMap[s] ?? 'badge-ghost'; }
function jobBadgeClass(s: string) { return jobBadgeMap[s] ?? 'badge-ghost'; }

async function fetchRun() {
	try {
		await execute(async () => { const data = await pipelineRunApi.get(runId); run.value = data; });
	} catch { toast.error('获取 Run 信息失败'); router.push('/ci/runs'); }
}

async function fetchJobs() {
	try { await executeJobs(async () => { jobs.value = await pipelineRunApi.listJobs(runId); }); }
	catch { /* silent */ }
}

async function fetchArtifacts() {
	try { await executeArtifacts(async () => { artifacts.value = await pipelineRunApi.listArtifacts(runId); }); }
	catch { /* silent */ }
}

async function showJobLogs(job: Job) {
	currentJob.value = job; showLogsDrawer.value = true; logs.value = null;
	try { await executeLogs(async () => { logs.value = await jobApi.listLogs(job.id); }); }
	catch { toast.error('获取日志失败'); }
}

async function handleRetry() {
	try {
		const newRun = await pipelineRunApi.retry(runId);
		toast.success('重试成功');
		router.push(`/ci/runs/${newRun.id}`);
	} catch (error) { toast.error(error instanceof Error ? error.message : '重试失败'); }
}

async function handleCancel() {
	try {
		await pipelineRunApi.cancel(runId);
		toast.success('已取消'); cancelModalRef.value?.close(); fetchRun();
	} catch (error) { toast.error(error instanceof Error ? error.message : '取消失败'); }
}

onMounted(() => { fetchRun(); fetchJobs(); fetchArtifacts(); });
</script>
