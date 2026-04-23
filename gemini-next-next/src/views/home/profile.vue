<template>
  <PageHeader
    :title="$t('common.profile.title')"
    :sub-title="$t('common.profile.subtitle')"
  ></PageHeader>
  <a-card>
    <a-row type="flex" justify="center" align="middle">
      <a-col :span="10">
        <a-form
          ref="formRef"
          :model="formItem"
          layout="vertical"
          :rules="rules"
        >
          <a-form-item :label="$t('user.form.user')">
            <a-input
              v-model:value="store.state.user.account.user"
              disabled
            ></a-input>
          </a-form-item>
          <a-form-item :label="$t('user.form.dept')">
            <a-input v-model:value="formItem.department"></a-input>
          </a-form-item>
          <a-form-item :label="$t('user.form.real_name')">
            <a-input v-model:value="formItem.real_name"></a-input>
          </a-form-item>
          <a-form-item :label="$t('user.form.email')">
            <a-input v-model:value="formItem.email"></a-input>
          </a-form-item>
          <a-form-item :label="$t('common.theme')">
            <a-select v-model:value="formItem.theme" @change="changeTheme">
              <a-select-option key="dark" value="dark"
                >{{ $t('common.theme.dark') }}
              </a-select-option>
              <a-select-option key="light" value="light"
                >{{ $t('common.theme.light') }}
              </a-select-option>
            </a-select>
          </a-form-item>
          <a-form-item :label="$t('user.password.new')" name="password">
            <a-input-password
              v-model:value="formItem.password"
            ></a-input-password>
          </a-form-item>
          <a-form-item
            :label="$t('user.password.confirm')"
            name="confirm_password"
          >
            <a-input-password
              v-model:value="formItem.confirm_password"
            ></a-input-password>
          </a-form-item>
          <a-form-item>
            <a-button
              block
              type="primary"
              @click="() => updateUserInfo(formItem, false)"
              >{{ $t('common.save') }}</a-button
            >
          </a-form-item>
        </a-form>

        <a-divider />

        <a-card :title="$t('profile.mfa.title')" size="small">
          <template v-if="!mfaEnabled && !mfaSetupMode">
            <a-alert
              :message="$t('profile.mfa.disabled')"
              type="warning"
              show-icon
              style="margin-bottom: 16px"
            />
            <a-button type="primary" @click="setupMFA">{{
              $t('profile.mfa.enable')
            }}</a-button>
          </template>

          <template v-if="mfaSetupMode">
            <a-alert
              :message="$t('profile.mfa.scan.hint')"
              type="info"
              show-icon
              style="margin-bottom: 16px"
            />
            <div style="text-align: center; margin-bottom: 16px">
              <img
                v-if="mfaQrCode"
                :src="mfaQrCode"
                alt="QR Code"
                width="200"
                height="200"
              />
            </div>
            <a-form-item :label="$t('profile.mfa.secret')">
              <a-input :value="mfaSecret" readonly />
            </a-form-item>
            <a-form-item :label="$t('profile.mfa.code')">
              <a-input
                v-model:value="mfaVerifyCode"
                :maxlength="6"
                :placeholder="$t('profile.mfa.code.placeholder')"
              />
            </a-form-item>
            <a-space>
              <a-button type="primary" @click="verifyMFA">{{
                $t('profile.mfa.confirm')
              }}</a-button>
              <a-button @click="mfaSetupMode = false">{{
                $t('common.cancel')
              }}</a-button>
            </a-space>
          </template>

          <template v-if="mfaEnabled && !mfaSetupMode">
            <a-alert
              :message="$t('profile.mfa.enabled')"
              type="success"
              show-icon
              style="margin-bottom: 16px"
            />
            <a-form-item :label="$t('profile.mfa.code')">
              <a-input
                v-model:value="mfaDisableCode"
                :maxlength="6"
                :placeholder="$t('profile.mfa.disable.placeholder')"
              />
            </a-form-item>
            <a-button danger @click="disableMFA">{{
              $t('profile.mfa.disable')
            }}</a-button>
          </template>
        </a-card>
      </a-col>
    </a-row>
  </a-card>
</template>

<script lang="ts" setup>
  import { useStore } from '@/store';
  import { onMounted, ref } from 'vue';
  import CommonMixins from '@/mixins/common';
  import PageHeader from '@/components/pageHeader/pageHeader.vue';
  import { RuleObject } from 'ant-design-vue/lib/form/interface';
  import {
    getUserInfo,
    RegisterForm,
    updateUserInfo,
    mfaSetup as mfaSetupApi,
    mfaVerify as mfaVerifyApi,
    mfaDisable as mfaDisableApi,
    mfaStatus,
  } from '@/apis/user';
  import { useI18n } from 'vue-i18n';
  import { message } from 'ant-design-vue';

  const store = useStore();

  const formItem = ref<RegisterForm>({
    password: '',
    confirm_password: '',
    real_name: '',
    email: '',
    department: '',
    theme: 'dark',
  });

  const mfaEnabled = ref(false);
  const mfaSetupMode = ref(false);
  const mfaQrCode = ref('');
  const mfaSecret = ref('');
  const mfaVerifyCode = ref('');
  const mfaDisableCode = ref('');

  const changeTheme = (e: any) => {
    localStorage.setItem('theme', e);
    location.reload();
  };

  const { t } = useI18n();

  const validPassword = async (rule: RuleObject, value: string) => {
    const pPattern = /^.*(?=.{6,})(?=.*\d)(?=.*[A-Z])(?=.*[a-z]).*$/;
    if (!pPattern.test(value)) {
      return Promise.reject(t('user.form.valid.password'));
    }
    if (value !== formItem.value.password && value !== '') {
      return Promise.reject('输入的密码不一致');
    } else {
      return Promise.resolve();
    }
  };

  const { regExpPassword } = CommonMixins();

  const rules = {
    password: [{ validator: regExpPassword, trigger: 'blur', required: true }],
    confirm_password: [
      { trigger: 'blur', message: '请确认密码', required: true },
      { required: true, validator: validPassword, trigger: 'blur' },
    ],
  };

  const setupMFA = async () => {
    const { data } = await mfaSetupApi();
    if (data.code === 1200) {
      mfaQrCode.value = data.payload.qr_code;
      mfaSecret.value = data.payload.secret;
      mfaSetupMode.value = true;
    }
  };

  const verifyMFA = async () => {
    if (mfaVerifyCode.value.length !== 6) {
      message.error(t('profile.mfa.code.invalid'));
      return;
    }
    const { data } = await mfaVerifyApi(mfaVerifyCode.value);
    if (data.code === 1200) {
      mfaEnabled.value = true;
      mfaSetupMode.value = false;
      mfaVerifyCode.value = '';
    }
  };

  const disableMFA = async () => {
    if (mfaDisableCode.value.length !== 6) {
      message.error(t('profile.mfa.code.invalid'));
      return;
    }
    const { data } = await mfaDisableApi(mfaDisableCode.value);
    if (data.code === 1200) {
      mfaEnabled.value = false;
      mfaDisableCode.value = '';
    }
  };

  const fetchMFAStatus = async () => {
    const { data } = await mfaStatus();
    if (data.code === 1200) {
      mfaEnabled.value = data.payload.mfa_enabled;
    }
  };

  onMounted(async () => {
    const { data } = await getUserInfo();
    formItem.value = data.payload;
    localStorage.getItem('theme') === null
      ? (formItem.value.theme = 'dark')
      : (formItem.value.theme = localStorage.getItem('theme') as string);
    localStorage.getItem('lang') === null
      ? (formItem.value.lang = 'zh_CN')
      : (formItem.value.lang = localStorage.getItem('lang') as string);
    fetchMFAStatus();
  });
</script>
