<template>
	<div class="login-container">
		<a-card class="login-card" title="Pomelo Orbit">
			<a-form :model="form" :rules="rules" @finish="handleLogin">
				<a-form-item name="username">
					<a-input v-model:value="form.username" placeholder="用户名" size="large">
						<template #prefix>
							<UserOutlined />
						</template>
					</a-input>
				</a-form-item>
				<a-form-item name="password">
					<a-input-password v-model:value="form.password" placeholder="密码" size="large">
						<template #prefix>
							<LockOutlined />
						</template>
					</a-input-password>
				</a-form-item>
				<a-form-item>
					<a-button type="primary" html-type="submit" size="large" block :loading="loading">
						登录
					</a-button>
				</a-form-item>
			</a-form>
		</a-card>
	</div>
</template>

<script setup lang="ts">
import { message } from 'ant-design-vue';
import { reactive } from 'vue';
import { useRouter } from 'vue-router';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useAuthStore } from '@/stores/auth';

const router = useRouter();
const authStore = useAuthStore();
const { loading, execute } = useStatusAsync();

const form = reactive({
	username: '',
	password: '',
});

const rules = {
	username: [{ required: true, message: '请输入用户名' }],
	password: [{ required: true, message: '请输入密码' }],
};

async function handleLogin() {
	try {
		await execute(async () => {
			await authStore.login(form.username, form.password);
			message.success('登录成功');
			router.push('/');
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '登录失败');
	}
}
</script>

<style scoped>
.login-container {
	display: flex;
	justify-content: center;
	align-items: center;
	min-height: 100vh;
	background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
}

.login-card {
	width: 400px;
	box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);
}

.login-card :deep(.ant-card-head-title) {
	text-align: center;
	font-size: 24px;
	font-weight: bold;
}
</style>
