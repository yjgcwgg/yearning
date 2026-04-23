<template>
  <a-row :gutter="24" align="middle">
    <a-col :xs="24" :md="8">
      <a-select
        v-model:value="selected"
        show-search
        allow-clear
        placeholder="数据源搜索"
        style="width: 100%"
        :filter-option="filterOption"
        @change="handleChange"
      >
        <a-select-option value="all">{{
          $t('order.state.all')
        }}</a-select-option>
        <a-select-option
          v-for="i in options"
          :key="i.source"
          :value="i.source"
          >{{ i.source }}</a-select-option
        >
      </a-select>
    </a-col>
    <a-col v-if="props.type !== 'query'" :xs="24" :md="16" style="text-align: right">
      <a-space>
        <a-checkbox
          v-model:checked="batchMode"
        >{{ $t('order.apply.batch.mode') }}</a-checkbox>
        <template v-if="batchMode && selectedSources.length > 0">
          <a-tag color="blue">{{ $t('order.apply.batch.selected', { count: selectedSources.length }) }}</a-tag>
          <a-button type="primary" size="small" @click="goBatchOrder">
            {{ $t('order.apply.batch.commit') }}
          </a-button>
          <a-button size="small" @click="selectedSources = []">
            {{ $t('order.apply.batch.clear') }}
          </a-button>
        </template>
      </a-space>
    </a-col>
  </a-row>
  <br />
  <a-list
    :loading="loading"
    :data-source="source"
    :grid="{ gutter: 16, xs: 1, sm: 1, md: 2, lg: 2, xl: 4, xxl: 4, xxxl: 4 }"
    :pagination="pagination"
  >
    <template #renderItem="{ item }">
      <a-list-item>
        <div
          @click="batchMode ? toggleSelect(item) : goSingleOrder(item)"
        >
          <a-card
            :body-style="{ paddingBottom: 20 }"
            hoverable
            :class="{ 'batch-selected': batchMode && isSelected(item) }"
            :style="batchMode && isSelected(item) ? { borderColor: '#1890ff', borderWidth: '2px' } : {}"
          >
            <a-card-meta :title="item.source">
              <template #description>{{
                $t('order.apply.card.env', {
                  env: item.idc,
                })
              }}</template>
              <template #avatar>
                <div style="position: relative">
                  <a-avatar :style="{ backgroundColor: batchMode && isSelected(item) ? '#1890ff' : '#Ff9900' }">
                    <template #icon>
                      <CodepenCircleOutlined />
                    </template>
                  </a-avatar>
                  <a-checkbox
                    v-if="batchMode"
                    :checked="isSelected(item)"
                    style="position: absolute; top: -4px; left: -4px"
                    @click.stop="toggleSelect(item)"
                  />
                </div>
              </template>
            </a-card-meta>
            <template #actions>
              <a-tooltip
                :title="
                  $t('order.apply.tab.source_id', { env: item.source_id })
                "
              >
                <SubnodeOutlined />
              </a-tooltip>
              <a-tooltip :title="$t('order.apply.card.env', { env: item.idc })">
                <ShareAltOutlined />
              </a-tooltip>
              <a-dropdown>
                <a-tooltip :title="$t('order.apply.card.enter')">
                  <a
                    class="ant-dropdown-link"
                    @click.stop="goSingleOrder(item)"
                  >
                    <EnterOutlined />
                  </a>
                </a-tooltip>
              </a-dropdown>
            </template>
          </a-card>
        </div>
      </a-list-item>
    </template>
  </a-list>
</template>

<script lang="ts" setup>
  import {
    SubnodeOutlined,
    EnterOutlined,
    ShareAltOutlined,
    CodepenCircleOutlined,
  } from '@ant-design/icons-vue';
  import { onMounted, ref } from 'vue';
  import { useRouter } from 'vue-router';
  import { useI18n } from 'vue-i18n';
  import { ISource, querySourceList } from '@/apis/source';

  const { t } = useI18n();

  const props = defineProps<{
    type: string;
    id: number;
    isExport?: boolean;
  }>();

  // eslint-disable-next-line @typescript-eslint/no-unused-vars
  const emit = defineEmits(['enter']);

  const router = useRouter();

  const pagination = {
    pageSize: 20,
  };

  const batchMode = ref(false);
  const selectedSources = ref<ISource[]>([]);

  const isSelected = (item: ISource) =>
    selectedSources.value.some((s) => s.source_id === item.source_id);

  const toggleSelect = (item: ISource) => {
    if (isSelected(item)) {
      selectedSources.value = selectedSources.value.filter(
        (s) => s.source_id !== item.source_id
      );
    } else {
      selectedSources.value = [...selectedSources.value, item];
    }
  };

  const goSingleOrder = (item: ISource) => {
    router.push({
      path: props.type !== 'query' ? '/apply/order' : '/apply/query',
      query: {
        type: props.id,
        idc: item.idc,
        source: item.source,
        source_id: item.source_id,
      },
    });
  };

  const goBatchOrder = () => {
    if (selectedSources.value.length === 0) return;
    const primary = selectedSources.value[0];
    const batchIds = selectedSources.value.slice(1).map((s) => s.source_id);
    router.push({
      path: '/apply/order',
      query: {
        type: props.id,
        idc: primary.idc,
        source: primary.source,
        source_id: primary.source_id,
        batch_ids: batchIds.join(','),
      },
    });
  };

  const filterOption = (input: string, option: any) => {
    return option.value.toLowerCase().indexOf(input.toLowerCase()) >= 0;
  };

  const handleChange = (value: string) => {
    value === '' || value === undefined || value === 'all'
      ? (source.value = tmpSource)
      : (source.value = tmpSource.filter(
          (item: ISource) => item.source === value
        ));
  };

  const selected = ref('all');

  let tmpSource = [] as ISource[];

  const source = ref([] as ISource[]);

  const options = ref([] as ISource[]);

  const loading = ref(true);

  onMounted(async () => {
    try {
      const { data } = await querySourceList(props.type);
      tmpSource = source.value = options.value = data.payload as ISource[];
      loading.value = false;
    } catch (error) {
      console.log(error);
    }
  });
</script>
