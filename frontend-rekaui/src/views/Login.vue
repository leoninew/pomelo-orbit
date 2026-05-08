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

<template>
	<div class="min-h-screen flex items-center justify-center bg-gradient-to-br from-primary/10 via-background to-accent/10 p-4">
		<div class="w-full max-w-md">
			<div class="bg-card rounded-lg border border-border shadow-lg p-8">
				<div class="text-center mb-8">
					<h1 class="text-3xl font-bold text-foreground mb-2">Pomelo Orbit</h1>
					<p class="text-muted-foreground">持续集成与持续部署平台</p>
				</div>

				<form class="space-y-4" @submit.prevent="handleLogin">
					<div class="space-y-2">
						<label for="username" class="text-sm font-medium text-foreground">用户名</label>
						<input
							id="username"
							v-model="form.username"
							type="text"
							class="w-full px-3 py-2 bg-background border rounded-md text-sm outline-none transition-colors focus:border-ring focus:ring-2 focus:ring-ring/20"
							:class="errors.username ? 'border-destructive' : 'border-input'"
							placeholder="请输入用户名"
							@input="errors.username = ''"
						/>
						<p v-if="errors.username" class="text-xs text-destructive">{{ errors.username }}</p>
					</div>

					<div class="space-y-2">
						<label for="password" class="text-sm font-medium text-foreground">密码</label>
						<div class="relative">
							<input
								id="password"
								v-model="form.password"
								:type="showPassword ? 'text' : 'password'"
								class="w-full px-3 py-2 pr-10 bg-background border rounded-md text-sm outline-none transition-colors focus:border-ring focus:ring-2 focus:ring-ring/20"
								:class="errors.password ? 'border-destructive' : 'border-input'"
								placeholder="请输入密码"
								@input="errors.password = ''"
								@keydown.enter="handleLogin"
							/>
							<button
								type="button"
								class="absolute right-2 top-1/2 -translate-y-1/2 p-1 text-muted-foreground hover:text-foreground transition-colors"
								@click="showPassword = !showPassword"
							>
								<Eye v-if="!showPassword" class="size-4" />
								<EyeOff v-else class="size-4" />
							</button>
						</div>
						<p v-if="errors.password" class="text-xs text-destructive">{{ errors.password }}</p>
					</div>

					<button
						type="submit"
						class="w-full px-4 py-2 bg-primary text-primary-foreground rounded-md font-medium hover:bg-primary/90 transition-colors disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center gap-2"
						:disabled="loading"
					>
						<span v-if="loading" class="inline-block size-4 border-2 border-primary-foreground/30 border-t-primary-foreground rounded-full animate-spin" />
						{{ loading ? '登录中...' : '登录' }}
					</button>
				</form>
			</div>
		</div>
	</div>
</template>
