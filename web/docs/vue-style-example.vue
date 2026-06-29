<template>
	<div class="container">
		<h1>{{ title }}</h1>
		<button @click="increment">Count: {{ count }}</button>
	</div>
</template>

<script setup lang="ts">
	import { ref } from 'vue';

	const title = ref('Hello World');
	const count = ref(0);

	function increment() {
		count.value++;
	}
</script>

<style scoped>
	.container {
		padding: 1rem;
		text-align: center;
	}

	button {
		margin-top: 1rem;
		padding: 0.5rem 1rem;
	}
</style>
