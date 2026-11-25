<template>
  <div class="rounded-2xl border border-gray-100 bg-white px-4 pb-4 pt-3 shadow-card dark:border-dark-700/50 dark:bg-dark-800/50">
    <!-- 一级:平台标签页 -->
    <div class="flex flex-wrap items-center gap-1 border-b border-gray-100 pb-2 dark:border-dark-700/60">
      <button
        v-for="p in ['all', ...platforms]"
        :key="`platform-${p}`"
        type="button"
        class="plaza-tab inline-flex items-center gap-1.5 rounded-lg px-3 py-1.5 text-sm font-medium transition disabled:cursor-not-allowed disabled:opacity-40"
        :class="platform === p ? 'plaza-tab-active' : 'text-gray-500 enabled:hover:bg-gray-50 enabled:hover:text-gray-900 dark:text-dark-400 dark:enabled:hover:bg-dark-800 dark:enabled:hover:text-white'"
        :disabled="p !== 'all' && !platformEnabled(p)"
        @click="$emit('update:platform', p)"
      >
        <PlatformIcon v-if="p !== 'all'" :platform="p as GroupPlatform" size="xs" />
        {{ p === 'all' ? t('modelPlaza.filters.all') : platformLabel(p) }}
      </button>
    </div>

    <!-- 二级:搜索 / 分组 / 倍率 / 视图切换 -->
    <div class="mt-3 flex flex-wrap items-center gap-2">
      <div class="relative min-w-[200px] flex-1">
        <Icon
          name="search"
          size="sm"
          class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-dark-500"
        />
        <input
          :value="search"
          type="text"
          :placeholder="t('modelPlaza.filters.searchPlaceholder')"
          class="input rounded-lg py-1.5 pl-9 pr-9"
          @input="$emit('update:search', ($event.target as HTMLInputElement).value)"
        />
        <button
          v-if="search"
          type="button"
          class="absolute right-2.5 top-1/2 -translate-y-1/2 text-gray-400 transition-colors hover:text-gray-600 dark:text-dark-500 dark:hover:text-gray-300"
          @click="$emit('update:search', '')"
        >
          <Icon name="x" size="xs" class="h-3.5 w-3.5" />
        </button>
      </div>

      <!-- 分组:当前平台/倍率下无结果的置灰 -->
      <select
        class="input w-full rounded-lg py-1.5 sm:w-auto sm:min-w-[220px] sm:flex-1"
        :value="String(groupId)"
        :aria-label="t('modelPlaza.filters.groupLabel')"
        @change="onGroupChange"
      >
        <option value="all">{{ t('modelPlaza.filters.allGroups') }}</option>
        <option
          v-for="g in groups"
          :key="`group-${g.id}`"
          :value="String(g.id)"
          :disabled="!groupEnabled(g)"
        >
          {{ g.name }}
        </option>
      </select>

      <!-- 倍率:当前平台/分组下不存在的置灰 -->
      <select
        class="input w-full rounded-lg py-1.5 sm:w-auto sm:min-w-[120px]"
        :value="String(rate)"
        :aria-label="t('modelPlaza.filters.rateLabel')"
        @change="onRateChange"
      >
        <option value="all">{{ t('modelPlaza.filters.allRates') }}</option>
        <option
          v-for="r in rates"
          :key="`rate-${r}`"
          :value="String(r)"
          :disabled="!rateEnabled(r)"
        >
          {{ r }}x
        </option>
      </select>

      <!-- 视图切换 -->
      <div class="inline-flex shrink-0 items-center rounded-lg p-0.5 ring-1 ring-inset ring-gray-200 dark:ring-dark-700">
        <button
          type="button"
          class="inline-flex h-8 w-8 items-center justify-center rounded-md transition"
          :class="viewMode === 'grid' ? activeViewClass : inactiveViewClass"
          :title="t('modelPlaza.filters.viewGrid')"
          :aria-label="t('modelPlaza.filters.viewGrid')"
          :aria-pressed="viewMode === 'grid'"
          @click="$emit('update:viewMode', 'grid')"
        >
          <Icon name="grid" size="sm" />
        </button>
        <button
          type="button"
          class="inline-flex h-8 w-8 items-center justify-center rounded-md transition"
          :class="viewMode === 'list' ? activeViewClass : inactiveViewClass"
          :title="t('modelPlaza.filters.viewList')"
          :aria-label="t('modelPlaza.filters.viewList')"
          :aria-pressed="viewMode === 'list'"
          @click="$emit('update:viewMode', 'list')"
        >
          <Icon name="menu" size="sm" />
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import { platformLabel } from '@/utils/platformColors'
import type { GroupPlatform } from '@/types'
import type { PlazaViewMode } from '@/api/modelPlaza'

const props = defineProps<{
  /** 数据中出现的平台(去重排序后)。 */
  platforms: string[]
  /** 全量分组(含平台与生效倍率),三个维度的置灰联动由此推导。 */
  groups: Array<{ id: number; name: string; platform: string; rate: number }>
  /** 全量生效倍率去重升序。 */
  rates: number[]
  platform: string
  groupId: number | 'all'
  rate: number | 'all'
  /** 模型名搜索词(纯前端过滤)。 */
  search: string
  viewMode: PlazaViewMode
}>()

const emit = defineEmits<{
  'update:platform': [value: string]
  'update:groupId': [value: number | 'all']
  'update:rate': [value: number | 'all']
  'update:search': [value: string]
  'update:viewMode': [value: PlazaViewMode]
}>()

const { t } = useI18n()

const activeViewClass = 'bg-primary-50 text-primary-600 dark:bg-primary-500/15 dark:text-primary-300'
const inactiveViewClass = 'text-gray-400 hover:text-gray-700 dark:text-dark-500 dark:hover:text-gray-200'

function onGroupChange(e: Event) {
  const v = (e.target as HTMLSelectElement).value
  emit('update:groupId', v === 'all' ? 'all' : Number(v))
}

function onRateChange(e: Event) {
  const v = (e.target as HTMLSelectElement).value
  emit('update:rate', v === 'all' ? 'all' : Number(v))
}

/**
 * 三个维度互为约束(faceted):某选项可选 ⟺ 在「其他两维」当前选择下仍有分组命中。
 * 「全部」永远可选,作为解除本维约束的出口。
 */
function platformEnabled(p: string): boolean {
  return props.groups.some(
    (g) =>
      g.platform === p &&
      (props.groupId === 'all' || g.id === props.groupId) &&
      (props.rate === 'all' || g.rate === props.rate)
  )
}

function groupEnabled(g: { platform: string; rate: number }): boolean {
  return (
    (props.platform === 'all' || g.platform === props.platform) &&
    (props.rate === 'all' || g.rate === props.rate)
  )
}

function rateEnabled(r: number): boolean {
  return props.groups.some(
    (g) =>
      g.rate === r &&
      (props.platform === 'all' || g.platform === props.platform) &&
      (props.groupId === 'all' || g.id === props.groupId)
  )
}
</script>

<style scoped>
.plaza-tab {
  position: relative;
}

.plaza-tab-active {
  @apply bg-primary-50 text-primary-700 dark:bg-primary-500/15 dark:text-primary-300;
}

.plaza-tab-active::after {
  content: '';
  position: absolute;
  left: 0.75rem;
  right: 0.75rem;
  bottom: -0.55rem;
  height: 2px;
  border-radius: 9999px;
  @apply bg-primary-500;
}
</style>
