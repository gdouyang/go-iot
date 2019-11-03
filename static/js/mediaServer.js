define(
function() {
  return {
    components:{
    },
    data: function () {
      return {
        tableData: [],
      }
    },
    mounted(){
      this.searchList();
    },
    methods: {
      searchList(){
        this.$refs.mainTable.search();
      },
      stopAll(){
        this.$confirm('此操作将中断所有客户的画面, 是否继续?', '提示', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }).then(() => {
          fetch('/mediasrs/stopall', {
            method: 'PUT',
          }).then(res => {
            return res.json()
          }).then(data => {
            this.searchList();
            this.$message({
              type: data.success ? 'success' : 'error',
              message: data.msg
           });
          })
        })
      },
      startAll(){
          fetch('/mediasrs/startall', {
            method: 'PUT',
          }).then(res => {
            return res.json()
          }).then(data => {
            this.searchList();
            this.$message({
              type: data.success ? 'success' : 'error',
              message: data.msg
           });
        })
      },
    },
    template: `
      <el-card class="box-card">
      <div slot="header" class="clearfix">
        <span>live media service</span>
        <el-button type="text" @click="searchList">刷新</el-button>
        <el-button type="text" @click="startAll">全部启动</el-button>
        <el-button type="text" @click="stopAll">全部停止</el-button>
      </div>
      <div class="text item">
        <my-table url="/mediasrs/list" ref="mainTable">
          <el-table-column prop="id" label="ID" width="70"/>
          <el-table-column prop="type" label="服务" min-width="120"/>
          <el-table-column prop="port" label="端口"/>
          <el-table-column prop="status" label="开启状态">
            <template slot-scope="scope">
              <el-tag :type="scope.row.status == 'on' ? '在线': '离线'" size="mini">
              {{scope.row.status}}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" :width="200" fixed="right">
            <template slot-scope="scope">
              <el-button @click="openDialog(scope.row, true)" type="text" size="small">编辑</el-button>
              <el-button type="text" size="small" @click="deleteRecord(scope.row)">停止</el-button>
            </template>
          </el-table-column>
        </my-table>
      </div>
      </el-card>
    ` 
  }
});