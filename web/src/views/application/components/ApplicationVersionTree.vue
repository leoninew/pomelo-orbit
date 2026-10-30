<template>
  <ul
    ref="tree"
    role="tree"
    :aria-label="t('application.detail.explorer.title')"
    class="space-y-1 py-3 text-sm"
  >
    <li
      v-for="version in versions"
      :key="version.id"
      role="treeitem"
      :data-node-key="version.id"
      :tabindex="focusKey === version.id ? 0 : -1"
      :aria-label="version.label"
      :aria-selected="isSelected(version.id)"
      :aria-expanded="expandedId === version.id"
      class="rounded-md outline-none focus-visible:ring-2 focus-visible:ring-ring"
      @focus.stop="focusedKey = version.id"
      @keydown.stop="handleKeydown($event, version.id)"
    >
      <div
        class="flex min-w-0 cursor-pointer items-center gap-2 rounded-md px-2 py-2 hover:bg-accent"
        :class="isSelected(version.id) ? 'bg-accent text-accent-foreground' : 'text-foreground'"
        @click="selectVersion(version.id)"
      >
        <button
          type="button"
          tabindex="-1"
          class="app-icon-button size-6 shrink-0"
          :aria-label="
            expandedId === version.id
              ? t('application.detail.actions.collapseComponents')
              : t('application.detail.actions.expandComponents')
          "
          @click.stop="emit('toggle', version.id)"
        >
          <ChevronRight
            class="size-4 transition-transform"
            :class="expandedId === version.id ? 'rotate-90' : ''"
          />
        </button>
        <Layers class="size-4 shrink-0 text-muted-foreground" />
        <span class="min-w-0 flex-1 break-all">{{ version.label }}</span>
      </div>
      <ul
        v-if="expandedId === version.id"
        role="group"
        class="ml-5 mt-1 space-y-1 border-l border-border pl-3"
      >
        <li v-if="loadStates[version.id] === 'loading'" role="none">
          <AppLoadingState size="compact" />
        </li>
        <li v-else-if="loadStates[version.id] === 'error'" role="none" class="px-2 py-2">
          <p class="text-destructive">{{ t('application.versionDetail.componentsLoadFailed') }}</p>
          <button class="app-link mt-2" @click="emit('retry', version.id)">
            {{ t('common.retry') }}
          </button>
        </li>
        <template v-else-if="details[version.id]">
          <li
            v-for="component in details[version.id]?.components"
            :key="component.id"
            role="treeitem"
            :data-node-key="componentKey(version.id, component.id)"
            :tabindex="focusKey === componentKey(version.id, component.id) ? 0 : -1"
            :aria-label="component.name"
            :aria-selected="isSelected(version.id, component.id)"
            class="flex cursor-pointer items-center gap-2 rounded-md px-3 py-2 outline-none hover:bg-accent focus-visible:ring-2 focus-visible:ring-ring"
            :class="
              isSelected(version.id, component.id)
                ? 'bg-accent text-accent-foreground'
                : 'text-foreground'
            "
            @click.stop="selectComponent(version.id, component.id)"
            @focus.stop="focusedKey = componentKey(version.id, component.id)"
            @keydown.stop="handleKeydown($event, version.id, component.id)"
          >
            <Box class="size-4 shrink-0 text-muted-foreground" />
            <span class="min-w-0 flex-1 break-all">{{ component.name }}</span>
          </li>
          <li
            v-if="details[version.id]?.components.length === 0"
            role="none"
            class="px-3 py-2 text-muted-foreground"
          >
            {{ t('application.versionDetail.empty.components') }}
          </li>
          <li v-if="version.status === 'unpublished'" role="none" class="px-3 py-2">
            <button
              type="button"
              class="app-link inline-flex items-center gap-2"
              :disabled="busy"
              @click.stop="emit('create-component', version.id)"
              @keydown.stop
            >
              <Plus class="size-4" />
              {{ t('application.detail.actions.addComponent') }}
            </button>
          </li>
        </template>
      </ul>
    </li>
  </ul>
</template>

<script setup lang="ts">
  import { Box, ChevronRight, Layers, Plus } from '@lucide/vue';
  import { computed, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import AppLoadingState from '@/components/ui/AppLoadingState.vue';
  import type { VersionResp } from '@/gen/proto/orbit/v1/application/version';
  import type { VersionTreeSelection, VersionTreeLoadState } from '../versionTree';

  const props = defineProps<{
    versions: VersionResp[];
    details: Record<string, VersionResp>;
    loadStates: Record<string, VersionTreeLoadState>;
    expandedId: string;
    selection: VersionTreeSelection | undefined;
    busy: boolean;
  }>();
  const emit = defineEmits<{
    select: [selection: VersionTreeSelection];
    toggle: [versionId: string];
    retry: [versionId: string];
    'create-component': [versionId: string];
  }>();
  const { t } = useI18n();
  const tree = ref<HTMLUListElement>();
  const focusedKey = ref('');
  const visibleKeys = computed(() =>
    props.versions.flatMap((version) => [
      version.id,
      ...(props.expandedId === version.id
        ? (props.details[version.id]?.components ?? []).map((component) =>
            componentKey(version.id, component.id)
          )
        : []),
    ])
  );
  const focusKey = computed(() => {
    if (visibleKeys.value.includes(focusedKey.value)) {
      return focusedKey.value;
    }
    const selected = props.selection;
    const selectedKey = selected?.componentId
      ? componentKey(selected.versionId, selected.componentId)
      : selected?.versionId;
    return selectedKey && visibleKeys.value.includes(selectedKey)
      ? selectedKey
      : visibleKeys.value[0];
  });

  function componentKey(versionId: string, componentId: string) {
    return `${versionId}/${componentId}`;
  }

  function isSelected(versionId: string, componentId?: string) {
    return props.selection?.versionId === versionId && props.selection.componentId === componentId;
  }

  function selectVersion(versionId: string) {
    focusedKey.value = versionId;
    emit('select', { versionId });
  }

  function selectComponent(versionId: string, componentId: string) {
    focusedKey.value = componentKey(versionId, componentId);
    emit('select', { versionId, componentId });
  }

  function focusNode(key: string | undefined) {
    if (!key) {
      return;
    }
    focusedKey.value = key;
    const nodes = tree.value?.querySelectorAll<HTMLElement>('[data-node-key]');
    Array.from(nodes ?? [])
      .find((node) => node.dataset.nodeKey === key)
      ?.focus();
  }

  function handleKeydown(event: KeyboardEvent, versionId: string, componentId?: string) {
    const key = componentId ? componentKey(versionId, componentId) : versionId;
    const index = visibleKeys.value.indexOf(key);
    switch (event.key) {
      case 'ArrowDown':
        focusNode(visibleKeys.value[index + 1]);
        break;
      case 'ArrowUp':
        focusNode(visibleKeys.value[index - 1]);
        break;
      case 'Home':
        focusNode(visibleKeys.value[0]);
        break;
      case 'End':
        focusNode(visibleKeys.value[visibleKeys.value.length - 1]);
        break;
      case 'ArrowRight':
        if (!componentId) {
          if (props.expandedId !== versionId) {
            emit('toggle', versionId);
          } else if (props.details[versionId]?.components.length) {
            focusNode(visibleKeys.value[index + 1]);
          }
        }
        break;
      case 'ArrowLeft':
        if (componentId) {
          focusNode(versionId);
        } else if (props.expandedId === versionId) {
          emit('toggle', versionId);
        }
        break;
      case 'Enter':
      case ' ':
        if (componentId) {
          selectComponent(versionId, componentId);
        } else {
          selectVersion(versionId);
        }
        break;
      default:
        return;
    }
    event.preventDefault();
  }
</script>
