<script lang="js">
import { post } from '~/utils/request'
export default {
  name: 'MyTable',
  props:{
    url: "",
    selectable: false
  },
  data(){
    return {
      tableData: [],
      page:{
          pageNum: 1,
          pageSize: 10,
          condition: {}
      },
      total: 0,
      loading: false,
      selectionData:[]
    }
  },
  methods:{
    search(param){
      if(param) {
          this.page.condition = param;
      }
      this.loading = true;
      post(this.url, this.page)
      .then(res =>  res.json(), err => this.loading = false)
      .then(data => {
        this.loading = false;
        console.log(data)
        if(data.list == null){
          data.list = []
        }
        this.$emit('done-load', data.list);
        this.tableData = data.list;
        this.total = data.total;
      }, err => this.loading = false)
    },
    clearSelection(){
      this.$refs.table.clearSelection()
      this.selectionData = []
    },
    selectionChange(selection){
      this.selectionData = selection;
    },
    getCheckData(key){
      var result = []
      this.selectionData.forEach(element => {
        if(key){
          result.push(element[key]);
        }else{
          result.push(element);
        }
      });
      return result;
    }
  }
}
</script>
<template>
  <div>
    <el-table :data="tableData" style="width: 100%" ref="table" v-loading="loading" 
      @selection-change="selectionChange">
      <el-table-column type="selection" width="55" v-if="selectable"/>
      <slot></slot>
    </el-table>
    <el-pagination small layout="prev, pager, next, total" :total="total" 
      :current-page.sync="page.pageNum" :page-size="page.pageSize" @current-change="search()">
    </el-pagination>
    <el-button type="text" icon="el-icon-refresh" @click="search()"></el-button>
  </div>
</template>

<style>
.el-button {
  margin: 4px;
}
</style>
