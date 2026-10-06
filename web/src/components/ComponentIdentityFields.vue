<template>
  <div class="space-y-4">
    <div class="space-y-1.5">
      <label :for="`${id}-user`" class="app-field-label block">
        {{ t('application.componentDetail.fields.user') }}
      </label>
      <input
        :id="`${id}-user`"
        v-model="user"
        class="app-input"
        :class="userError ? 'app-input-error' : ''"
        :aria-invalid="userError ? 'true' : undefined"
        :aria-describedby="userError ? `${id}-user-error` : undefined"
        type="text"
      />
      <p v-if="userError" :id="`${id}-user-error`" class="app-field-error" role="alert">
        {{ t('application.componentDetail.validation.user') }}
      </p>
    </div>
    <div class="space-y-1.5">
      <label :for="`${id}-groups`" class="app-field-label block">
        {{ t('application.componentDetail.fields.groupAdd') }}
      </label>
      <input
        :id="`${id}-groups`"
        v-model="groups"
        class="app-input"
        :class="groupsError ? 'app-input-error' : ''"
        :aria-invalid="groupsError ? 'true' : undefined"
        :aria-describedby="groupsError ? `${id}-groups-error` : undefined"
        type="text"
      />
      <p v-if="groupsError" :id="`${id}-groups-error`" class="app-field-error" role="alert">
        {{ t('application.componentDetail.validation.groupAdd') }}
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, useId } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { parseGroupAdd, validComponentUser } from '@/utils/componentIdentity';

  const user = defineModel<string>('user', { required: true });
  const groups = defineModel<string>('groups', { required: true });
  const id = useId();
  const { t } = useI18n();
  const userError = computed(() => !validComponentUser(user.value));
  const groupsError = computed(() => parseGroupAdd(groups.value) === null);
</script>
