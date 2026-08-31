<script setup lang="ts">
import _ from 'lodash-es'
import request from '@/axios'
import { PropType, ref } from 'vue'
import Table from './Table.vue'
import type { TableColumn } from './types'
const props = defineProps({
  url: {
    type: String,
    default: () => ''
  },
  // 表头
  columns: {
    type: Array as PropType<TableColumn[]>,
    default: () => []
  },
  pageNum: {
    type: Number,
    default: 1
  },
  pageSize: {
    type: Number,
    default: 10
  }
})
const emits = defineEmits(['update:pageNum', 'update:pageSize'])
const tableRef = ref(null)
const pageNumRef = ref(props.pageNum)
const pageSizeRef = ref(props.pageSize)
const loading = ref(true)
const total = ref(0)
const totalPage = ref(0)

const tableDataList = ref<[]>([])
const cache = {
  searchAfterList: [],
  searchAfter: null,
  currentSearchAfter: null,
  pageNum: props.pageNum,
  pageSize: props.pageSize,
  condition: null
}

const getTableList = async (pageNum: number, pageSize: number, params?: any) => {
  if (params !== undefined) {
    cache.condition = params
  }
  const p = {
    pageNum: pageNum,
    pageSize: pageSize,
    searchAfter: null,
    condition: cache.condition
  }
  if (p.pageNum === 1 || (cache.pageSize && pageSize !== cache.pageSize)) {
    p.searchAfter = null
    cache.searchAfterList = []
    cache.searchAfter = null
  } else if (cache.searchAfter) {
    if (pageNum < cache.pageNum) {
      cache.searchAfterList = _.dropRight(cache.searchAfterList, 2)
      cache.searchAfter = _.last(cache.searchAfterList)
    }
    p.searchAfter = cache.searchAfter
  }
  cache.currentSearchAfter = p.searchAfter
  cache.pageNum = pageNum
  cache.pageSize = pageSize
  loading.value = true
  const res = await request.post(props.url, p).finally(() => {
    loading.value = false
  })
  if (res) {
    const r = res.result || {}
    // 无数据时也要渲染空表，避免 list 为 null/undefined 导致表格区域异常
    tableDataList.value = r.list || []
    total.value = r.totalCount || 0
    totalPage.value = r.totalPage || 0
    if (r.searchAfter && !_.isEmpty(r.searchAfter)) {
      cache.searchAfterList.push(r.searchAfter)
      cache.searchAfter = r.searchAfter
    }
  }
}

const search = (params?: any) => {
  pageNumRef.value = props.pageNum
  getTableList(pageNumRef.value, pageSizeRef.value, params)
}
const refresh = async () => {
  const p = {
    pageNum: cache.pageNum,
    pageSize: cache.pageSize,
    searchAfter: cache.currentSearchAfter,
    condition: cache.condition
  }
  loading.value = true
  const res = await request.post(props.url, p).finally(() => {
    loading.value = false
  })
  if (res) {
    const r = res.result || {}
    tableDataList.value = r.list || []
    total.value = r.totalCount || 0
    totalPage.value = r.totalPage || 0
  }
}
const pageSizeChange = (val: number) => {
  if (pageSizeRef.value === val) {
    return
  }
  emits('update:pageSize', val)
  emits('update:pageNum', 1)
  pageNumRef.value = 1
  pageSizeRef.value = val
  getTableList(1, val, cache.condition)
}
const pageNumChange = (val: number) => {
  if (pageNumRef.value === val) {
    return
  }
  emits('update:pageNum', val)
  pageNumRef.value = val
  getTableList(val, pageSizeRef.value, cache.condition)
}

const getSelectedRowKeys = (key: string) => {
  const rows = tableRef?.value?.elTableRef?.getSelectionRows()
  if (key) {
    return rows.map((row: any) => row[key])
  }
  return rows
}

defineExpose({
  search: search,
  refresh: refresh,
  getSelectedRows: getSelectedRowKeys
})
</script>

<template>
  <Table
    ref="tableRef"
    :pageSize="pageSizeRef"
    :currentPage="pageNumRef"
    :columns="columns"
    :data="tableDataList"
    :loading="loading"
    :max-height="650"
    :pagination="{
      total: total,
      pageCount: totalPage,
      layout: 'sizes, prev, next, slot, total',
      prevText: '上一页',
      nextText: '下一页'
    }"
    @update:pageSize="pageSizeChange"
    @update:currentPage="pageNumChange"
    v-bind="$attrs"
  />
</template>

<style lang="less" scoped>
:deep(.el-table) {
  --el-table-header-bg-color: var(--el-fill-color-light);
}
:deep(.el-table th.el-table__cell) {
  background-color: var(--el-table-header-bg-color);
  color: var(--el-text-color-regular);
  font-weight: 500;
}
:deep(.el-pagination) {
  justify-content: flex-end;
  .el-pager {
    li {
      display: none;
      &:last-child {
        display: flex;
      }
    }
    .is-active {
      display: flex;
    }
  }
}
</style>
