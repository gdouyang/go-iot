Vue.component('my-table', {
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
      selectionData:[]
    }
  },
  methods:{
    search(param){
      if(param) {
          this.page.condition = param;
      }
      fetch(this.url, {
        method: 'POST',
        body: JSON.stringify(this.page),
        headers: new Headers({
          'Content-Type': 'application/json'
        })
      }).then(res => {
        return res.json()
      }).then(data => {
        console.log(data)
        if(data.list == null){
          data.list = []
        }
        this.tableData = data.list;
        this.total = data.total;
      })
    },
    clearSelection(){
      this.$refs.table.clearSelection()
      this.selectionData = []
    },
    selectionChange(selection){
      this.selectionData = selection;
    },
    getSelectData(key){
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
  },
	template: `
    <div>
      <el-table :data="tableData" style="width: 100%" ref="table" @selection-change="selectionChange">
        <el-table-column type="selection" width="55" v-if="selectable"/>
        <slot></slot>
      </el-table>
      <el-pagination layout="prev, pager, next" :total="total" 
        :current-page.sync="page.pageNum" :page-size="page.pageSize" @current-change="search()">
      </el-pagination>
    </div>
	`
});
