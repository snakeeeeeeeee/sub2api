<template>
  <section class="border-t border-gray-200 pt-4 dark:border-dark-600">
    <button
      type="button"
      class="flex w-full items-center justify-between gap-3 text-left"
      :aria-expanded="expanded"
      data-testid="upstream-headers-toggle"
      @click="expanded = !expanded"
    >
      <span>
        <span class="block text-sm font-medium text-gray-700 dark:text-gray-300">
          上游 Header 模板
        </span>
        <span class="mt-1 block text-xs text-gray-500 dark:text-gray-400">
          显式配置要发送给上游的 Header，支持从入站 header、query、body 和账号字段取值。
        </span>
      </span>
      <Icon
        name="chevronDown"
        size="md"
        :class="['shrink-0 text-gray-400 transition-transform', expanded && 'rotate-180']"
      />
    </button>

    <div v-if="expanded" class="mt-4 space-y-3" data-testid="upstream-headers-editor">
      <div
        v-if="rows.length === 0"
        class="rounded-lg border border-dashed border-gray-300 px-3 py-4 text-sm text-gray-500 dark:border-dark-500 dark:text-gray-400"
      >
        未配置自定义上游 Header。
      </div>

      <div v-else class="space-y-2">
        <div
          class="hidden grid-cols-1 gap-2 px-1 text-xs font-medium text-gray-500 dark:text-gray-400 sm:grid sm:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)_44px]"
        >
          <span>上游 Header</span>
          <span>值 / 模板</span>
          <span class="sr-only">操作</span>
        </div>
        <div
          v-for="(row, index) in rows"
          :key="row.id"
          class="grid grid-cols-1 gap-2 sm:grid-cols-[minmax(0,0.8fr)_minmax(0,1.2fr)_44px]"
          data-testid="upstream-header-row"
        >
          <input
            v-model="row.name"
            :ref="(el) => setNameInputRef(el, row.id)"
            type="text"
            class="input font-mono text-sm"
            placeholder="例如 X-Pool-Session-ID"
            data-testid="upstream-header-name"
            @input="emitModelValue"
          />
          <div class="flex min-w-0 gap-2">
            <input
              v-model="row.value"
              type="text"
              class="input min-w-0 flex-1 font-mono text-sm"
              placeholder="固定值或 {{header.session-id}}"
              data-testid="upstream-header-value"
              @focus="activeRowIndex = index"
              @input="emitModelValue"
            />
            <select
              class="h-11 w-36 rounded-lg border border-gray-300 bg-white px-2 text-xs text-gray-700 dark:border-dark-500 dark:bg-dark-700 dark:text-gray-200"
              aria-label="插入变量"
              data-testid="upstream-header-variable"
              @change="insertVariable(index, $event)"
            >
              <option value="">插入变量</option>
              <option v-for="item in variableOptions" :key="item.value" :value="item.value">
                {{ item.label }}
              </option>
            </select>
          </div>
          <button
            type="button"
            class="flex h-11 w-11 items-center justify-center rounded-lg text-red-500 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20"
            aria-label="删除 Header"
            data-testid="upstream-header-remove"
            @click="removeRow(index)"
          >
            <Icon name="trash" size="sm" :stroke-width="2" />
          </button>
        </div>
      </div>

      <div class="flex flex-wrap items-center gap-2">
        <button
          type="button"
          class="btn btn-secondary text-sm"
          data-testid="upstream-header-add"
          @click="addRow"
        >
          <Icon name="plus" size="sm" :stroke-width="2" />
          添加 Header
        </button>
        <button
          v-for="preset in presetRows"
          :key="preset.name"
          type="button"
          class="rounded-lg bg-gray-100 px-3 py-1.5 text-xs font-medium text-gray-600 transition-colors hover:bg-gray-200 dark:bg-dark-600 dark:text-gray-300 dark:hover:bg-dark-500"
          @click="addPreset(preset)"
        >
          + {{ preset.name }}
        </button>
      </div>

      <p class="text-xs leading-5 text-gray-500 dark:text-gray-400">
        Header 名和值/模板都可手动输入，变量下拉只负责插入占位符片段。缺值时该 Header 会跳过；Authorization、Cookie、Host、Content-Type、x-api-key 等敏感或协议头不会被覆盖。
      </p>
    </div>
  </section>
</template>

<script setup lang="ts">
import { nextTick, ref, watch, type ComponentPublicInstance } from 'vue'
import Icon from '@/components/icons/Icon.vue'

type HeaderMap = Record<string, string>

interface HeaderRow {
  id: number
  name: string
  value: string
}

const props = defineProps<{
  modelValue?: HeaderMap
}>()

const emit = defineEmits<{
  'update:modelValue': [value: HeaderMap]
}>()

const expanded = ref(false)
const activeRowIndex = ref(0)
const rows = ref<HeaderRow[]>([])
const nameInputRefs = new Map<number, HTMLInputElement>()
let nextRowID = 1
let skipNextModelSync = false

const variableOptions = [
  { label: 'header.session-id', value: '{{header.session-id}}' },
  { label: 'query.session_id', value: '{{query.session_id}}' },
  { label: 'body.metadata.tenant_id', value: '{{body.metadata.tenant_id}}' },
  { label: 'json_header.x-newapi-meta:user.id', value: '{{json_header.x-newapi-meta:user.id}}' },
  { label: 'account.id', value: '{{account.id}}' },
  { label: 'account.name', value: '{{account.name}}' },
  { label: 'account.platform', value: '{{account.platform}}' },
  { label: 'account.extra.pool.id', value: '{{account.extra.pool.id}}' }
]

const presetRows = [
  { name: 'X-Pool-Session-ID', value: '{{header.session-id}}' },
  { name: 'X-Tenant-ID', value: '{{body.metadata.tenant_id}}' },
  { name: 'X-User-ID', value: '{{json_header.x-newapi-meta:user.id}}' },
  { name: 'X-Account-ID', value: '{{account.id}}' }
]

const normalizeModelValue = (value?: HeaderMap): HeaderRow[] => {
  if (!value) return []
  return Object.entries(value)
    .filter(([name, template]) => name.trim() && typeof template === 'string')
    .map(([name, template]) => ({
      id: nextRowID++,
      name,
      value: template
    }))
}

const toHeaderMap = () => {
  const out: HeaderMap = {}
  for (const row of rows.value) {
    const name = row.name.trim()
    if (!name) continue
    out[name] = row.value
  }
  return out
}

const emitModelValue = () => {
  skipNextModelSync = true
  emit('update:modelValue', toHeaderMap())
}

const addRow = () => {
  const row = { id: nextRowID++, name: '', value: '' }
  rows.value.push(row)
  activeRowIndex.value = rows.value.length - 1
  emitModelValue()
  void nextTick(() => {
    nameInputRefs.get(row.id)?.focus()
  })
}

const removeRow = (index: number) => {
  rows.value.splice(index, 1)
  activeRowIndex.value = Math.max(0, Math.min(activeRowIndex.value, rows.value.length - 1))
  emitModelValue()
}

const addPreset = (preset: { name: string; value: string }) => {
  const existing = rows.value.find((row) => row.name.trim().toLowerCase() === preset.name.toLowerCase())
  if (existing) {
    existing.value = preset.value
  } else {
    rows.value.push({ id: nextRowID++, name: preset.name, value: preset.value })
    activeRowIndex.value = rows.value.length - 1
  }
  emitModelValue()
}

const insertVariable = (index: number, event: Event) => {
  const select = event.target as HTMLSelectElement
  const value = select.value
  select.value = ''
  if (!value) return
  const row = rows.value[index]
  if (!row) return
  row.value = row.value ? `${row.value}${value}` : value
  activeRowIndex.value = index
  emitModelValue()
}

const setNameInputRef = (el: Element | ComponentPublicInstance | null, rowID: number) => {
  if (el instanceof HTMLInputElement) {
    nameInputRefs.set(rowID, el)
  } else {
    nameInputRefs.delete(rowID)
  }
}

watch(
  () => props.modelValue,
  (value) => {
    if (skipNextModelSync) {
      skipNextModelSync = false
      return
    }
    rows.value = normalizeModelValue(value)
    if (rows.value.length > 0) {
      expanded.value = true
    }
  },
  { immediate: true }
)
</script>
