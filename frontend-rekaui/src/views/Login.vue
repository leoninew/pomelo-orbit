<template>
	<div
		class="min-h-screen flex items-center justify-center bg-gradient-to-br from-primary/10 via-background to-accent/10 p-4"
	>
		<div class="w-full max-w-md">
			<div class="app-surface rounded-lg p-8">
				<div class="text-center mb-8">
					<h1 class="text-3xl font-bold text-foreground mb-2">Pomelo Orbit</h1>
					<p class="text-muted-foreground">持续集成与持续部署平台</p>
				</div>

				<form class="space-y-4" @submit.prevent="handleLogin">
					<div class="space-y-1.5">
						<label for="username" class="app-field-label block">用户名</label>
						<input
							id="username"
							v-model="form.username"
							type="text"
							class="app-input"
							:class="errors.username ? 'app-input-error' : ''"
							placeholder="请输入用户名"
							@input="errors.username = ''"
						/>
						<p v-if="errors.username" class="app-field-error text-xs">{{ errors.username }}</p>
					</div>

					<div class="space-y-1.5">
						<label for="password" class="app-field-label block">密码</label>
						<div class="relative">
							<input
								id="password"
								v-model="form.password"
								:type="showPassword ? 'text' : 'password'"
								class="app-input pr-10"
								:class="errors.password ? 'app-input-error' : ''"
								placeholder="请输入密码"
								@input="errors.password = ''"
								@keydown.enter="handleLogin"
							/>
							<button
								type="button"
								class="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-muted-foreground transition-colors hover:text-foreground"
								@click="showPassword = !showPassword"
							>
								<Eye v-if="!showPassword" class="size-4" />
								<EyeOff v-else class="size-4" />
							</button>
						</div>
						<p v-if="errors.password" class="app-field-error text-xs">{{ errors.password }}</p>
					</div>

					<button
						type="submit"
						class="app-button-primary flex w-full items-center justify-center gap-2"
						:disabled="loading"
					>
						<span
							v-if="loading"
							class="inline-block size-4 animate-spin rounded-full border-2 border-primary-foreground/30 border-t-primary-foreground"
						/>
						{{ loading ? '登录中...' : '登录' }}
					</button>
				</form>
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">
	import { Eye, EyeOff } from 'lucide-vue-next';
	import { reactive, ref } from 'vue';
	import { useRouter } from 'vue-router';
	import { useAuthStore } from '@/stores/auth';
	import { useToast } from '@/composables/useToast';

	const router = useRouter();
	const authStore = useAuthStore();
	const toast = useToast();

	const form = reactive({
		username: '',
		password: '',
	});

	const errors = reactive({
		username: '',
		password: '',
	});

	const showPassword = ref(false);
	const loading = ref(false);

	function validate() {
		errors.username = form.username.trim() ? '' : '请输入用户名';
		errors.password = form.password.trim() ? '' : '请输入密码';
		return !errors.username && !errors.password;
	}

	async function handleLogin() {
		if (!validate()) {
			return;
		}

		loading.value = true;
		try {
			await authStore.login(form.username, form.password);
			toast.success('登录成功');
			router.push('/');
		} catch (err) {
			toast.error(err instanceof Error ? err.message : '登录失败');
		} finally {
			loading.value = false;
		}
	}
</script>
