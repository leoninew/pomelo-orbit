/**
 * 变量来源标签管理
 *
 * 统一管理变量来源的显示标签和样式
 */

export type VariableSource =
	| 'global'
	| 'repository'
	| 'repository_custom'
	| 'template'
	| 'template_stage'
	| 'template_custom'
	| 'runtime';

/**
 * 获取变量来源的显示标签
 *
 * 规则：
 * - 仓库详情界面：repository -> "项目运行时"，repository_custom -> "项目自定义"
 * - 模板详情界面：template -> "模板运行时"，template_stage -> "模板 Stage"，template_custom -> "模板自定义"
 */
export function getSourceLabel(source: VariableSource): string {
	const labels: Record<string, string> = {
		global: '全局',
		repository: '项目运行时',
		repository_custom: '项目自定义',
		template: '模板运行时',
		template_stage: '模板 Stage',
		template_custom: '模板自定义',
		runtime: '触发时',
	};
	return labels[source] || source;
}

/**
 * 获取变量来源的徽章样式
 */
export function getSourceBadgeClass(source: VariableSource): string {
	const classes: Record<string, string> = {
		global: 'border border-violet-200 bg-violet-50 text-violet-700',
		repository: 'border border-blue-200 bg-blue-50 text-blue-700',
		repository_custom: 'border border-amber-200 bg-amber-50 text-amber-700',
		template: 'border border-primary/20 bg-primary/10 text-primary',
		template_stage: 'border border-green-200 bg-green-50 text-green-700',
		template_custom: 'border border-amber-200 bg-amber-50 text-amber-700',
		runtime: 'border border-border bg-muted text-muted-foreground',
	};
	return classes[source] || 'border border-border bg-muted text-muted-foreground';
}

/**
 * 判断变量是否可编辑
 * template_stage 变量（从 Stage 脚本提取）可在触发时覆盖其 default 值
 */
export function isVariableEditable(source: VariableSource): boolean {
	return (
		source === 'repository_custom' || source === 'template_custom' || source === 'template_stage'
	);
}
