<template>
  <article
    class="plaza-card flex flex-col rounded-2xl border bg-white p-4 shadow-card transition hover:-translate-y-0.5 hover:shadow-lg dark:bg-dark-800/50"
    :style="accentStyle"
  >
    <!-- 头部:平台图标 / 模型名 / 平台 / 复制 -->
    <header class="flex items-start gap-3">
      <div class="plaza-card-icon flex h-10 w-10 shrink-0 items-center justify-center rounded-xl">
        <PlatformIcon :platform="model.platform as GroupPlatform" size="md" />
      </div>
      <div class="min-w-0 flex-1">
        <h3 class="break-all text-base font-semibold leading-6 text-gray-900 dark:text-white">{{ model.name }}</h3>
        <p class="text-xs text-gray-500 dark:text-dark-400">{{ platformLabel(model.platform) }}</p>
      </div>
      <button
        type="button"
        class="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-md text-gray-400 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:text-dark-400 dark:hover:bg-dark-700 dark:hover:text-primary-400"
        :title="t('modelPlaza.table.copyModelId', { model: model.name })"
        :aria-label="t('modelPlaza.table.copyModelId', { model: model.name })"
        @click="copyToClipboard(model.name)"
      >
        <Icon name="copy" size="sm" aria-hidden="true" />
      </button>
    </header>

    <p class="mt-3 text-sm text-gray-600 dark:text-dark-300">
      {{ t('modelPlaza.card.via', { platform: platformLabel(model.platform) }) }}
    </p>

    <!-- 标签:计费方式 / 分组 / 生效倍率 -->
    <div class="mt-3 flex min-w-0 items-center gap-1.5 text-xs">
      <span class="shrink-0 rounded-md bg-gray-100 px-1.5 py-0.5 text-gray-600 dark:bg-dark-700/70 dark:text-dark-300">
        {{ billingModeLabel }}
      </span>
      <span
        class="min-w-0 flex-1 truncate rounded-md bg-gray-50 px-1.5 py-0.5 text-gray-600 dark:bg-dark-700/40 dark:text-dark-300"
        :title="group.name"
      >
        {{ group.name }}
      </span>
      <span v-if="model.time_pricing" class="shrink-0 rounded-md bg-amber-50 px-1.5 py-0.5 text-amber-700 dark:bg-amber-900/20 dark:text-amber-300">
        {{ t('modelPlaza.card.timePricing') }}
      </span>
      <span class="plaza-card-rate shrink-0 rounded-md px-1.5 py-0.5 font-mono font-semibold">{{ rowRate }}x</span>
    </div>

    <!-- 价格 -->
    <div class="mt-3 border-t border-gray-100 pt-3 dark:border-dark-700/60">
      <!-- token 计费:输入 / 输出 / 缓存 -->
      <div v-if="isToken" class="grid grid-cols-3 gap-2 text-xs">
        <div>
          <div class="mb-1 text-gray-500 dark:text-dark-400">{{ t('modelPlaza.table.input') }}</div>
          <PriceLine :official="officialPrice(model.official_pricing?.input_price)" :paid="paid(model.pricing?.input_price)" />
        </div>
        <div>
          <div class="mb-1 text-gray-500 dark:text-dark-400">{{ t('modelPlaza.table.output') }}</div>
          <PriceLine :official="officialPrice(model.official_pricing?.output_price)" :paid="paid(model.pricing?.output_price)" />
        </div>
        <div>
          <div class="mb-1 text-gray-500 dark:text-dark-400">{{ t('modelPlaza.table.cache') }}</div>
          <template v-if="hasCache">
            <div class="flex flex-wrap items-baseline gap-x-1">
              <span class="text-gray-500 dark:text-dark-400">{{ t('modelPlaza.table.cacheWriteShort') }}</span>
              <PriceLine :official="officialPrice(model.official_pricing?.cache_write_price)" :paid="paid(model.pricing?.cache_write_price)" />
            </div>
            <div v-if="model.pricing?.cache_write_1h_price != null" class="flex flex-wrap items-baseline gap-x-1">
              <span class="text-gray-500 dark:text-dark-400">{{ t('modelPlaza.table.cacheWriteShort') }} 1h</span>
              <PriceLine :official="officialPrice(model.official_pricing?.cache_write_1h_price)" :paid="paid(model.pricing.cache_write_1h_price)" />
            </div>
            <div class="flex flex-wrap items-baseline gap-x-1">
              <span class="text-gray-500 dark:text-dark-400">{{ t('modelPlaza.table.cacheReadShort') }}</span>
              <PriceLine :official="officialPrice(model.official_pricing?.cache_read_price)" :paid="paid(model.pricing?.cache_read_price)" />
            </div>
          </template>
          <span v-else class="text-gray-400 dark:text-dark-500">-</span>
        </div>
      </div>

      <!-- 按次 / 按图 / 视频:单价或档位 -->
      <div v-else class="flex flex-wrap items-center gap-1.5 text-xs">
        <template v-if="requestIntervals.length">
          <span
            v-for="(iv, idx) in requestIntervals"
            :key="idx"
            class="inline-flex items-center gap-1 rounded-md bg-gray-100 px-2 py-0.5 font-mono text-gray-800 dark:bg-dark-700/60 dark:text-gray-200"
          >
            <span class="font-sans text-gray-400 dark:text-dark-500">{{ iv.tier_label }}</span>
            {{ paidRequest(iv.per_request_price) }}<span class="font-sans text-gray-400 dark:text-dark-500">{{ perUnitSuffix }}</span>
          </span>
        </template>
        <template v-else-if="model.pricing?.per_request_price != null">
          <span class="font-mono text-sm font-semibold text-gray-900 dark:text-gray-50">{{ paidRequest(model.pricing.per_request_price) }}</span>
          <span class="text-gray-400 dark:text-dark-500">{{ perUnitSuffix }}</span>
        </template>
        <span v-else class="text-gray-400 dark:text-dark-500">-</span>
      </div>
    </div>

    <!-- 底部:官方参考价 / 查看详情 -->
    <div class="min-h-3 flex-1"></div>
    <footer class="flex items-end justify-between gap-2 border-t border-gray-100 pt-3 text-xs dark:border-dark-700/60">
      <div class="min-w-0 text-gray-500 dark:text-dark-400">
        <template v-if="isToken && model.official_pricing">
          <div>
            {{ t('modelPlaza.card.officialRef') }}
            {{ officialPrice(model.official_pricing.input_price) }} / {{ officialPrice(model.official_pricing.output_price) }}
          </div>
          <div class="mt-0.5 text-gray-400 dark:text-dark-500">{{ t('modelPlaza.table.unitPerMillion') }}</div>
        </template>
        <span v-else class="text-gray-400 dark:text-dark-500">{{ t('modelPlaza.card.noOfficial') }}</span>
      </div>
      <button
        type="button"
        class="plaza-card-link inline-flex shrink-0 items-center gap-0.5 font-semibold transition"
        @click="$emit('detail')"
      >
        {{ t('modelPlaza.card.viewDetail') }}
        <Icon name="arrowRight" size="xs" class="h-3.5 w-3.5" />
      </button>
    </footer>
  </article>
</template>

<script setup lang="ts">
import { computed, defineComponent, h } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { useClipboard } from '@/composables/useClipboard'
import { formatScaled } from '@/utils/pricing'
import { platformAccentColor, platformLabel } from '@/utils/platformColors'
import {
  BILLING_MODE_IMAGE,
  BILLING_MODE_TOKEN,
  BILLING_MODE_VIDEO,
  type BillingMode
} from '@/constants/channel'
import type { ModelPlazaGroup, PlazaModel } from '@/api/modelPlaza'
import type { GroupPlatform } from '@/types'

const props = defineProps<{
  model: PlazaModel
  group: ModelPlazaGroup
}>()

defineEmits<{ detail: [] }>()

const { t } = useI18n()
const { copyToClipboard } = useClipboard()

const PER_MILLION = 1_000_000

/** 划线官方价 + 加粗实付价;两者相同或官方缺失时只显示实付。 */
const PriceLine = defineComponent({
  props: { official: { type: String, default: '-' }, paid: { type: String, default: '-' } },
  setup(p) {
    return () =>
      h('span', { class: 'inline-flex flex-wrap items-baseline gap-x-1 font-mono' }, [
        p.official !== '-' && p.official !== p.paid
          ? h('span', { class: 'text-gray-400 line-through dark:text-dark-500' }, p.official)
          : null,
        h('span', { class: 'font-semibold text-gray-900 dark:text-gray-50' }, p.paid)
      ])
  }
})

const accentStyle = computed(() => ({ '--card-accent': platformAccentColor(props.model.platform) }))

const mode = computed<BillingMode>(() => (props.model.pricing?.billing_mode || BILLING_MODE_TOKEN) as BillingMode)
const isToken = computed(() => mode.value === BILLING_MODE_TOKEN)

const billingModeLabel = computed(() => {
  if (mode.value === BILLING_MODE_TOKEN) return t('modelPlaza.card.tokenBilling')
  if (mode.value === BILLING_MODE_IMAGE) return t('modelPlaza.table.perImage')
  return t('modelPlaza.table.perRequest')
})

/** 生效倍率 = 用户专属倍率 ?? 分组默认倍率。 */
const effectiveRate = computed(() => props.group.user_rate_multiplier ?? props.group.rate_multiplier)

/** 本卡片的展示倍率:图片/视频独立倍率开启时取独立倍率,与计费口径一致。 */
const rowRate = computed(() => {
  if (mode.value === BILLING_MODE_VIDEO && props.group.video_rate_independent) {
    return Math.max(0, props.group.video_rate_multiplier ?? 1)
  }
  if (mode.value === BILLING_MODE_IMAGE && props.group.image_rate_independent) {
    return props.group.image_rate_multiplier ?? 1
  }
  return effectiveRate.value
})

function paid(value: number | null | undefined): string {
  if (value == null) return '-'
  return formatScaled(value * effectiveRate.value, PER_MILLION, 2)
}

function officialPrice(value: number | null | undefined): string {
  if (value == null) return '-'
  return formatScaled(value, PER_MILLION, 2)
}

function paidRequest(value: number | null | undefined): string {
  if (value == null) return '-'
  return formatScaled(value * rowRate.value, 1, 2)
}

const hasCache = computed(() => {
  const p = props.model.pricing
  return p?.cache_write_price != null || p?.cache_write_1h_price != null || p?.cache_read_price != null
})

const requestIntervals = computed(() =>
  (props.model.pricing?.intervals ?? []).filter((iv) => iv.per_request_price != null)
)

const perUnitSuffix = computed(() =>
  mode.value === BILLING_MODE_IMAGE ? t('modelPlaza.table.perUnitImage') : t('modelPlaza.table.perUnitRequest')
)
</script>

<style scoped>
.plaza-card {
  border-color: color-mix(in srgb, var(--card-accent) 28%, transparent);
}

.plaza-card:hover {
  border-color: color-mix(in srgb, var(--card-accent) 55%, transparent);
}

.plaza-card-icon {
  color: var(--card-accent);
  background-color: color-mix(in srgb, var(--card-accent) 10%, transparent);
  box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--card-accent) 22%, transparent);
}

.plaza-card-rate {
  color: color-mix(in srgb, var(--card-accent) 80%, black);
  background-color: color-mix(in srgb, var(--card-accent) 10%, transparent);
}

.dark .plaza-card-rate {
  color: color-mix(in srgb, var(--card-accent) 70%, white);
}

.plaza-card-link {
  @apply text-primary-700 hover:text-primary-800 dark:text-primary-300 dark:hover:text-primary-200;
}
</style>
