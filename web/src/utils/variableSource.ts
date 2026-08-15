/**
 * 变量来源标签管理
 *
 * 统一管理变量来源的显示标签和样式
 */

import type { BadgeTone } from './status';

export type VariableSource =
  | 'global'
  | 'repository'
  | 'repository_custom'
  | 'pipeline'
  | 'pipeline_custom'
  | 'pipeline_stage'
  | 'template'
  | 'template_stage'
  | 'template_custom'
  | 'runtime'
  | 'system';

/**
 * 获取变量来源的显示标签
 *
 * 规则：
 * - 仓库详情界面：repository -> "项目运行时"，repository_custom -> "项目自定义"
 * - 模板详情界面：template -> "模板运行时"，template_stage -> "模板 Stage"，template_custom -> "模板自定义"
 */
export function getSourceLabel(source: VariableSource): string {
  const labels: Record<VariableSource, string> = {
    global: '全局',
    repository: '项目运行时',
    repository_custom: '项目自定义',
    pipeline: '流水线配置',
    pipeline_custom: '流水线自定义',
    pipeline_stage: '流水线 Stage',
    template: '模板运行时',
    template_stage: '模板 Stage',
    template_custom: '模板自定义',
    runtime: '触发时',
    system: '系统',
  };
  return labels[source];
}

/**
 * 获取变量来源的徽章样式
 */
export function getSourceTone(source: string): BadgeTone {
  const tones: Record<VariableSource, BadgeTone> = {
    global: 'primary',
    repository: 'info',
    repository_custom: 'warning',
    pipeline: 'info',
    pipeline_custom: 'warning',
    pipeline_stage: 'success',
    template: 'primary',
    template_stage: 'success',
    template_custom: 'warning',
    runtime: 'default',
    system: 'default',
  };
  return tones[source as VariableSource] ?? 'default';
}

/**
 * 判断变量是否可编辑
 * template_stage 变量（从 Stage 脚本提取）可在触发时覆盖其 default 值
 */
export function isVariableEditable(source: string): boolean {
  return (
    source === 'repository_custom' ||
    source === 'pipeline_custom' ||
    source === 'template_custom' ||
    source === 'template_stage'
  );
}
