<template>
  <div ref="container">
    <a-card>
      <order-table-search
        ref="search"
        @search="
          (exp) => {
            tblRef.expr = exp;
            tbl.manual();
          }
        "
      >
      </order-table-search>

      <div
        v-if="isAudit === 'audit' && selectedRowKeys.length > 0"
        style="
          margin-bottom: 16px;
          padding: 8px 16px;
          background: #e6f7ff;
          border: 1px solid #91d5ff;
          border-radius: 4px;
          display: flex;
          align-items: center;
          gap: 12px;
        "
      >
        <span>{{ $t('order.batch.selected', { count: selectedRowKeys.length }) }}</span>
        <a-button type="primary" size="small" @click="batchAgree">{{
          $t('order.batch.agree')
        }}</a-button>
        <a-button danger size="small" @click="showBatchReject">{{
          $t('order.batch.reject')
        }}</a-button>
        <a-button size="small" @click="clearSelection">{{
          $t('order.batch.cancel')
        }}</a-button>
      </div>

      <c-table
        ref="tbl"
        :tbl-ref="tblRef"
        :size="props.size"
        :row-selection="
          isAudit === 'audit'
            ? {
                selectedRowKeys: selectedRowKeys,
                onChange: onSelectChange,
                getCheckboxProps: getCheckboxProps,
              }
            : undefined
        "
      >
        <template #bodyCell="{ column, text, record }">
          <template v-if="column.dataIndex === 'type'">
            <span>{{ text === 0 ? 'DDL' : 'DML' }}</span>
          </template>
          <template v-if="column.dataIndex === 'assigned'">
            <a-tag v-for="i in text.split(',')" :key="i">{{ i }}</a-tag>
          </template>
          <template v-if="column.dataIndex === 'delay'">{{
            text === 'none' ? $t('order.table.delay') : text
          }}</template>
          <template v-if="column.dataIndex === 'status'">
            <state-tags :state="text"></state-tags>
          </template>
          <template v-if="column.dataIndex === 'action'">
            <a-space>
              <a-button type="primary" size="small" @click="profile(record)"
                >{{ $t('common.profile') }}
              </a-button>
              <a-button
                v-if="
                  !props.disabledBtn &&
                  record.status === OrderState.PROCESS &&
                  record.delay !== 'none'
                "
                size="small"
                @click="delay.openSchedule(record.work_id)"
                >{{ $t('order.delay') }}</a-button
              >
            </a-space>
          </template>
        </template>
      </c-table>
    </a-card>
    <Profile
      :visible="visible"
      :width="width"
      @close="onClose"
      @close-drawer="() => (visible = false)"
    />
    <Delay ref="delay" />

    <a-modal
      v-model:visible="batchRejectVisible"
      :title="$t('order.batch.reject')"
      @ok="confirmBatchReject"
    >
      <a-form-item :label="$t('order.batch.reject.reason')">
        <a-textarea
          v-model:value="batchRejectText"
          :rows="3"
          :placeholder="$t('order.batch.reject.placeholder')"
        ></a-textarea>
      </a-form-item>
    </a-modal>
  </div>
</template>

<script lang="ts" setup>
  import StateTags from './stateTags.vue';
  import OrderTableSearch from './orderTableSearch.vue';
  import { onBeforeRouteUpdate, useRoute } from 'vue-router';
  import { onMounted, reactive, ref } from 'vue';
  import {
    OrderExpr,
    OrderParams,
    checkUri,
    batchAuditOrder,
  } from '@/apis/orderPostApis';
  import { OrderTableData, OrderState } from '@/types';
  import { useStore } from '@/store';
  import { useI18n } from 'vue-i18n';
  import { tableRef } from '@/components/table';
  import { useElementSize, useWebSocket } from '@vueuse/core';
  import { checkSchema } from '@/lib';
  import Profile from '@/components/orderProfile/index.vue';
  import Delay from './delay.vue';
  import { message, Modal } from 'ant-design-vue';

  interface propsAttr {
    size?: string;
    disabledBtn?: boolean;
  }

  const props = withDefaults(defineProps<propsAttr>(), {
    size: 'default',
    disabledBtn: false,
  });

  const search = ref();

  const delay = ref();

  const { t } = useI18n();

  const container = ref();

  const route = useRoute();

  const visible = ref<boolean>(false);

  const { width } = useElementSize(container);

  const store = useStore();

  const tbl = ref();

  const isAudit = ref('');

  const selectedRowKeys = ref<string[]>([]);

  const batchRejectVisible = ref(false);

  const batchRejectText = ref('');

  const onSelectChange = (keys: string[]) => {
    selectedRowKeys.value = keys;
  };

  const getCheckboxProps = (record: OrderTableData) => ({
    disabled:
      record.status !== 2 ||
      !record.assigned.split(',').includes(store.state.user.account.user),
  });

  const clearSelection = () => {
    selectedRowKeys.value = [];
  };

  const batchAgree = () => {
    Modal.confirm({
      title: t('order.batch.agree.confirm', {
        count: selectedRowKeys.value.length,
      }),
      onOk: async () => {
        const { data } = await batchAuditOrder({
          work_ids: selectedRowKeys.value,
          tp: 'agree',
        });
        const result = data.payload;
        const successCount = result.success ? result.success.length : 0;
        const failedCount = result.failed ? result.failed.length : 0;
        message.info(
          t('order.batch.result', {
            success: successCount,
            failed: failedCount,
          })
        );
        if (failedCount > 0) {
          result.failed.forEach((item) => {
            message.error(`${item.work_id}: ${item.error}`);
          });
        }
        clearSelection();
        tbl.value.manual();
      },
    });
  };

  const showBatchReject = () => {
    batchRejectText.value = '';
    batchRejectVisible.value = true;
  };

  const confirmBatchReject = async () => {
    const { data } = await batchAuditOrder({
      work_ids: selectedRowKeys.value,
      tp: 'reject',
      text: batchRejectText.value,
    });
    const result = data.payload;
    const successCount = result.success ? result.success.length : 0;
    const failedCount = result.failed ? result.failed.length : 0;
    message.info(
      t('order.batch.result', { success: successCount, failed: failedCount })
    );
    if (failedCount > 0) {
      result.failed.forEach((item) => {
        message.error(`${item.work_id}: ${item.error}`);
      });
    }
    batchRejectVisible.value = false;
    clearSelection();
    tbl.value.manual();
  };

  const tblRef = reactive<tableRef>({
    col: [
      {
        title: t('common.table.work_id'),
        dataIndex: 'work_id',
        width: 200,
      },
      {
        title: t('common.table.source'),
        dataIndex: 'source',
      },
      {
        title: t('common.table.remark'),
        dataIndex: 'text',
        ellipsis: true,
      },
      {
        title: t('common.table.type'),
        dataIndex: 'type',
      },
      {
        title: t('common.table.post.time'),
        dataIndex: 'date',
      },
      {
        title: t('common.table.post.user'),
        dataIndex: 'username',
      },
      {
        title: t('common.table.post.real_name'),
        dataIndex: 'real_name',
      },
      {
        title: t('order.profile.timing'),
        dataIndex: 'delay',
      },
      {
        title: t('order.profile.auditor'),
        dataIndex: 'assigned',
      },
      {
        title: t('common.table.state'),
        dataIndex: 'status',
      },
      {
        title: t('common.action'),
        dataIndex: 'action',
        width: 200,
      },
    ],
    data: [] as OrderTableData[],
    pageCount: 0,
    defaultPageSize: 20,
    expr: {
      status: 8,
      type: 2,
      text: '',
      username: '',
    } as OrderExpr,
    isloop: true,
    websocket: useWebSocket(
      `${checkSchema()}${checkUri(route.params.tp as string)}`,
      {
        autoReconnect: {
          retries: 3,
        },
        protocols: [store.state.user.account.token],
        onMessage: (e, event) => {
          let payload = JSON.parse(event.data);
          tblRef.data = payload.payload.data;
          tblRef.pageCount = payload.payload.page;
        },
      }
    ),
    fn: async (expr: OrderParams) => {
      tblRef.websocket?.send(JSON.stringify(expr));
    },
  });

  const profile = (record: OrderTableData) => {
    store.commit('order/ORDER_STORE', record);
    visible.value = true;
  };

  const onClose = () => {
    visible.value = false;
  };

  onBeforeRouteUpdate((to) => {
    isAudit.value = to.params.tp as string;
    tbl.value.manual();
  });

  onMounted(() => {
    isAudit.value = route.params.tp as string;
  });
</script>
