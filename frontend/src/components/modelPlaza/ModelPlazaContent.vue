<template>
  <div class="space-y-5">
    <!-- 页头(独立形态下展示标题;后台形态 AppHeader 已有页面标题) -->
    <div v-if="!embedded">
      <h1 class="text-2xl font-bold tracking-tight text-gray-900 dark:text-white sm:text-3xl">{{ t('modelPlaza.title') }}</h1>
      <p class="mt-1.5 text-sm text-gray-500 dark:text-dark-400">{{ t('modelPlaza.description') }}</p>
    </div>

    <!-- 全局价格说明(管理员配置,Markdown) -->
    <div
      v-if="descriptionHtml"
      class="plaza-description rounded-2xl border border-gray-100 bg-white px-5 py-4 text-sm shadow-card dark:border-dark-700/50 dark:bg-dark-800/50"
      v-html="descriptionHtml"
    ></div>

    <!-- 未登录提示 -->
    <p
      v-if="!isAuthenticated"
      class="flex items-center gap-1.5 text-xs text-gray-400 dark:text-dark-500"
    >
      <Icon name="infoCircle" size="xs" class="h-3.5 w-3.5" />
      {{ t('modelPlaza.anonymousHint') }}
    </p>

    <!-- 加载/错误/空 -->
    <div v-if="loading" class="flex min-h-[240px] items-center justify-center">
      <div class="h-8 w-8 animate-spin rounded-full border-2 border-primary-600/25 border-t-primary-600 dark:border-primary-400/25 dark:border-t-primary-400"></div>
    </div>
    <div
      v-else-if="error"
      class="rounded-2xl border border-red-200 bg-red-50 px-5 py-8 text-center text-sm text-red-600 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300"
    >
      {{ t('modelPlaza.loadFailed') }}
    </div>
    <template v-else>
      <!-- 统计条 -->
      <div
        class="flex flex-wrap items-center justify-between gap-2 rounded-2xl border border-primary-100/70 bg-gradient-to-r from-primary-50/80 to-white px-5 py-3 dark:border-primary-500/20 dark:from-primary-500/10 dark:to-dark-800/40"
      >
        <div class="flex flex-wrap items-baseline gap-x-5 gap-y-1 text-sm text-gray-500 dark:text-dark-400">
          <span><b class="mr-1 text-base text-gray-900 dark:text-white">{{ stats.models }}</b>{{ t('modelPlaza.stats.models') }}</span>
          <span><b class="mr-1 text-base text-gray-900 dark:text-white">{{ stats.platforms }}</b>{{ t('modelPlaza.stats.platforms') }}</span>
          <span><b class="mr-1 text-base text-gray-900 dark:text-white">{{ stats.groups }}</b>{{ t('modelPlaza.stats.groups') }}</span>
        </div>
        <span class="inline-flex items-center gap-1.5 text-xs text-primary-700 dark:text-primary-300">
          <span class="h-1.5 w-1.5 rounded-full bg-primary-500"></span>
          {{ t('modelPlaza.stats.autoSync') }}
        </span>
      </div>

      <!-- 筛选区:平台标签 + 搜索 / 分组 / 倍率 / 视图 -->
      <PlazaFilterBar
        :platforms="platforms"
        :groups="groupOptions"
        :rates="rates"
        :platform="selectedPlatform"
        :group-id="selectedGroupId"
        :rate="selectedRate"
        :search="searchQuery"
        :view-mode="viewMode"
        @update:platform="selectedPlatform = $event"
        @update:group-id="selectedGroupId = $event"
        @update:rate="selectedRate = $event"
        @update:search="searchQuery = $event"
        @update:view-mode="setViewMode"
      />

      <template v-if="filteredGroups.length > 0">
        <!-- 卡片视图:按分组倍率升序平铺所有 (分组, 模型) -->
        <div v-if="viewMode === 'grid'" class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
          <PlazaModelCard
            v-for="item in cardItems"
            :key="item.key"
            :model="item.model"
            :group="item.group"
            @detail="openDetail(item.group, item.model)"
          />
        </div>
        <!-- 列表视图:分组分节的价格表 -->
        <div v-else class="space-y-5">
          <PlazaGroupSection v-for="g in filteredGroups" :key="g.id" :group="g" />
        </div>
      </template>
      <div
        v-else
        class="rounded-2xl border border-dashed border-gray-300 px-5 py-12 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-dark-400"
      >
        {{ searchActive ? t('modelPlaza.noSearchResult') : t('modelPlaza.empty') }}
      </div>
    </template>

    <!-- 模型详情:复用分组价格表,只放当前模型 -->
    <BaseDialog
      :show="detail !== null"
      :title="detail?.model.name ?? ''"
      width="full"
      close-on-click-outside
      @close="detail = null"
    >
      <PlazaGroupSection v-if="detail" :group="detailGroup!" />
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import Icon from '@/components/icons/Icon.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import PlazaFilterBar from './PlazaFilterBar.vue'
import PlazaGroupSection from './PlazaGroupSection.vue'
import PlazaModelCard from './PlazaModelCard.vue'
import type { ModelPlazaGroup, ModelPlazaResponse, PlazaModel, PlazaViewMode } from '@/api/modelPlaza'
import { useAuthStore } from '@/stores/auth'

const props = defineProps<{
  response: ModelPlazaResponse | null
  loading: boolean
  error?: boolean
  /** 后台内嵌形态(AppLayout 内):隐藏页头。 */
  embedded?: boolean
}>()

const { t } = useI18n()
const authStore = useAuthStore()
const isAuthenticated = computed(() => authStore.isAuthenticated)

const VIEW_MODE_KEY = 'sub2api.modelPlaza.viewMode'

function loadViewMode(): PlazaViewMode {
  try {
    const v = localStorage.getItem(VIEW_MODE_KEY)
    if (v === 'grid' || v === 'list') return v
  } catch {
    /* 隐私模式等场景无 localStorage */
  }
  return 'grid'
}

const viewMode = ref<PlazaViewMode>(loadViewMode())

function setViewMode(mode: PlazaViewMode) {
  viewMode.value = mode
  try {
    localStorage.setItem(VIEW_MODE_KEY, mode)
  } catch {
    /* ignore */
  }
}

const selectedPlatform = ref<string>('all')
const selectedGroupId = ref<number | 'all'>('all')
const selectedRate = ref<number | 'all'>('all')
const searchQuery = ref('')

const searchActive = computed(() => searchQuery.value.trim() !== '')

const descriptionHtml = computed(() => {
  const md = props.response?.description?.trim()
  if (!md) return ''
  return DOMPurify.sanitize(marked.parse(md) as string)
})

/** 生效倍率 = 用户专属倍率 ?? 分组默认倍率。 */
function effectiveRate(g: ModelPlazaGroup): number {
  return g.user_rate_multiplier ?? g.rate_multiplier
}

const allGroups = computed(() => props.response?.groups ?? [])

/** 顶部统计:模型按名称去重,平台取模型实际平台(Composite 分组会展开)。 */
const stats = computed(() => {
  const models = new Set<string>()
  const platformSet = new Set<string>()
  for (const g of allGroups.value) {
    for (const m of g.models) {
      models.add(m.name)
      if (m.platform) platformSet.add(m.platform)
    }
  }
  return { models: models.size, platforms: platformSet.size, groups: allGroups.value.length }
})

const platforms = computed(() =>
  [...new Set(allGroups.value.map((g) => g.platform).filter(Boolean))].sort()
)

const groupOptions = computed(() =>
  allGroups.value.map((g) => ({
    id: g.id,
    name: g.name,
    platform: g.platform,
    rate: effectiveRate(g)
  }))
)

/** 全量生效倍率;当前组合下不可用的项由 FilterBar 置灰而非隐藏。 */
const rates = computed(() =>
  [...new Set(allGroups.value.map(effectiveRate))].sort((a, b) => a - b)
)

/** 数据刷新后选中的倍率可能不复存在,重置为全部。 */
watch(rates, (list) => {
  if (selectedRate.value !== 'all' && !list.includes(selectedRate.value)) {
    selectedRate.value = 'all'
  }
})

const filteredGroups = computed(() => {
  let groups = allGroups.value
  if (selectedPlatform.value !== 'all') {
    groups = groups.filter((g) => g.platform === selectedPlatform.value)
  }
  if (selectedGroupId.value !== 'all') {
    groups = groups.filter((g) => g.id === selectedGroupId.value)
  }
  if (selectedRate.value !== 'all') {
    groups = groups.filter((g) => effectiveRate(g) === selectedRate.value)
  }
  // 模型名搜索:分组内只留命中的模型,整组无命中则隐藏该分组。
  const q = searchQuery.value.trim().toLowerCase()
  if (q) {
    groups = groups
      .map((g) => ({ ...g, models: g.models.filter((m) => m.name.toLowerCase().includes(q)) }))
      .filter((g) => g.models.length > 0)
  }
  // 专属倍率会改变生效值,不能只依赖后端按默认倍率的排序。
  return [...groups].sort(
    (a, b) => effectiveRate(a) - effectiveRate(b) || a.name.localeCompare(b.name)
  )
})

/** 卡片视图条目:同一模型在不同分组各占一张卡(价格/倍率不同)。 */
const cardItems = computed(() =>
  filteredGroups.value.flatMap((g) =>
    g.models.map((m) => ({ key: `${g.id}:${m.platform}:${m.name}`, group: g, model: m }))
  )
)

const detail = ref<{ group: ModelPlazaGroup; model: PlazaModel } | null>(null)

function openDetail(group: ModelPlazaGroup, model: PlazaModel) {
  detail.value = { group, model }
}

const detailGroup = computed<ModelPlazaGroup | null>(() =>
  detail.value ? { ...detail.value.group, models: [detail.value.model] } : null
)
</script>

<style scoped>
.plaza-description {
  line-height: 1.7;
  overflow-wrap: anywhere;
}

.plaza-description :deep(h1),
.plaza-description :deep(h2),
.plaza-description :deep(h3) {
  @apply mb-2 mt-3 font-semibold text-gray-900 first:mt-0 dark:text-white;
}

.plaza-description :deep(p) {
  @apply mb-2 text-gray-700 last:mb-0 dark:text-dark-200;
}

.plaza-description :deep(a) {
  @apply text-primary-600 underline underline-offset-4 hover:text-primary-700 dark:text-primary-300;
}

.plaza-description :deep(ul) {
  @apply mb-2 list-disc pl-5;
}

.plaza-description :deep(ol) {
  @apply mb-2 list-decimal pl-5;
}

.plaza-description :deep(li) {
  @apply mb-0.5 text-gray-700 dark:text-dark-200;
}

.plaza-description :deep(code) {
  @apply rounded bg-gray-100 px-1.5 py-0.5 font-mono text-xs dark:bg-dark-800;
}

.plaza-description :deep(blockquote) {
  @apply my-2 border-l-4 border-gray-300 pl-3 text-gray-600 dark:border-dark-600 dark:text-dark-300;
}
</style>
