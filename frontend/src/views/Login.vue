<template>
	<div
		class="min-h-screen flex items-center justify-center bg-gradient-to-br from-indigo-500 to-purple-700"
	>
		<div class="card bg-base-100 w-96 shadow-2xl">
			<div class="card-body gap-6">
				<h1 class="text-2xl font-bold text-center">Pomelo Orbit</h1>

				<form class="flex flex-col gap-4" @submit.prevent="handleLogin">
					<!-- Username -->
					<fieldset class="fieldset">
						<legend class="fieldset-legend">用户名</legend>
						<label
							class="input w-full flex items-center gap-2"
							:class="{ 'input-error': errors.username }"
						>
							<UserRound class="size-4 text-base-content/60 shrink-0" />
							<input
								v-model="form.username"
								type="text"
								placeholder="请输入用户名"
								class="grow"
								autocomplete="username"
							/>
						</label>
						<p v-if="errors.username" class="fieldset-label text-error">{{ errors.username }}</p>
					</fieldset>

					<!-- Password -->
					<fieldset class="fieldset">
						<legend class="fieldset-legend">密码</legend>
						<label
							class="input w-full flex items-center gap-2"
							:class="{ 'input-error': errors.password }"
						>
							<Lock class="size-4 text-base-content/60 shrink-0" />
							<input
								v-model="form.password"
								:type="showPassword ? 'text' : 'password'"
								placeholder="请输入密码"
								class="grow"
								autocomplete="current-password"
							/>
							<button
								type="button"
								class="text-base-content/60 hover:text-base-content transition-colors"
								@click="showPassword = !showPassword"
							>
								<Eye v-if="!showPassword" class="size-4" />
								<EyeOff v-else class="size-4" />
							</button>
						</label>
						<p v-if="errors.password" class="fieldset-label text-error">{{ errors.password }}</p>
					</fieldset>

					<button type="submit" class="btn btn-primary w-full mt-2" :disabled="loading">
						<span v-if="loading" class="loading loading-spinner loading-sm" />
						登录
					</button>
				</form>
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">
import { Eye, EyeOff, Lock, UserRound } from 'lucide-vue-next';
import { reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import { useAuthStore } from '@/stores/auth';

const router = useRouter();
const authStore = useAuthStore();
const toast = useToast();
const { loading, execute } = useStatusAsync();

const showPassword = ref(false);
const form = reactive({ username: '', password: '' });
const errors = reactive({ username: '', password: '' });

function validate() {
	errors.username = form.username.trim() ? '' : '请输入用户名';
	errors.password = form.password ? '' : '请输入密码';
	return !errors.username && !errors.password;
}

async function handleLogin() {
	if (!validate()) {
		return;
	}
	try {
		await execute(async () => {
			await authStore.login(form.username, form.password);
			toast.success('登录成功');
			router.push('/');
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '登录失败');
	}
}
</script>
