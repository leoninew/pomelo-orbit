import { ref } from 'vue';
import { serviceApi } from '@/api/service/service';
import type { ServiceResp } from '@/gen/proto/orbit/v1/service/service';

const pageSize = 100;

export function useRouteTargetServices() {
  const services = ref<ServiceResp[]>([]);
  const loading = ref(false);

  async function load(projectId: string) {
    loading.value = true;
    try {
      const items: ServiceResp[] = [];
      let page = 1;
      let pages = 1;
      do {
        const response = await serviceApi.list({ project_id: projectId, page, per_page: pageSize });
        items.push(...response.items);
        pages = response.pages;
        page += 1;
      } while (page <= pages);
      services.value = items;
    } finally {
      loading.value = false;
    }
  }

  return { services, loading, load };
}
