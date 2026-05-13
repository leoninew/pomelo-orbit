<template>
	<div
		class="min-h-screen flex items-center justify-center bg-gradient-to-br from-primary/10 via-background to-accent/10"
	>
		<div class="text-center">
			<div
				class="inline-block animate-spin rounded-full h-12 w-12 border-4 border-primary border-t-transparent"
			></div>
			<p class="mt-4 text-muted-foreground">{{ message }}</p>
		</div>
	</div>
</template>

<script setup lang="ts">
	import { onMounted, ref } from 'vue';
	import { useRouter } from 'vue-router';
	import { useI18n } from 'vue-i18n';
	import { authApi } from '@/api/auth';
	import { useAuthStore } from '@/stores/auth';
	import { useToast } from '@/composables/useToast';

	const { t } = useI18n();
	const router = useRouter();
	const authStore = useAuthStore();
	const toast = useToast();
	const message = ref(t('login.googleProcessing'));

	onMounted(async () => {
		const params = new URLSearchParams(window.location.search);
		const code = params.get('code');
		const error = params.get('error');

		if (error || !code) {
			message.value = t('login.googleCancelled');
			setTimeout(() => router.push('/login'), 1500);
			return;
		}

		try {
			const response = await authApi.googleCallback({ code });
			authStore.setToken(response.access_token);
			const profile = await authApi.getCurrentUser();
			authStore.setUser(profile);
			toast.success(t('login.loginSuccess'));
			router.push('/');
		} catch (err: unknown) {
			message.value = err instanceof Error ? err.message : t('login.googleFailed');
			setTimeout(() => router.push('/login'), 2000);
		}
	});
</script>
