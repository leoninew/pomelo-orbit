import {
  inject,
  onBeforeUnmount,
  provide,
  shallowRef,
  toValue,
  watchEffect,
  type InjectionKey,
  type MaybeRefOrGetter,
  type ShallowRef,
} from 'vue';

export interface BreadcrumbItem {
  label?: string;
  labelKey?: string;
  to?: string;
}

const breadcrumbItemsKey = Symbol('breadcrumbItems') as InjectionKey<ShallowRef<BreadcrumbItem[]>>;

export function provideBreadcrumbItems() {
  const breadcrumbItems = shallowRef<BreadcrumbItem[]>([]);
  provide(breadcrumbItemsKey, breadcrumbItems);
  return breadcrumbItems;
}

export function usePageBreadcrumbs(items: MaybeRefOrGetter<BreadcrumbItem[]>) {
  const breadcrumbItems = inject(breadcrumbItemsKey);
  if (!breadcrumbItems) {
    throw new Error('Breadcrumb items were not provided.');
  }

  watchEffect(() => {
    breadcrumbItems.value = toValue(items);
  });

  onBeforeUnmount(() => {
    breadcrumbItems.value = [];
  });
}
