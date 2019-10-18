define(["components/device_add", "components/switchOpt", "components/lightOpt", "components/ledFileUpload"
, "components/screenshot"], 
function(deviceAdd, switchOpt, lightOpt, ledFileUpload, screenshot) {
  return {
    components:{
      'add-dialog': deviceAdd,
      'switch-opt': switchOpt,
      'light-opt': lightOpt,
      'led-file-upload': ledFileUpload,
      'screenshot': screenshot,
    },
    data: function () {
      return {
        tableData: [],
        searchParam:{id:''}
      }
    },
    mounted(){
      this.searchList();
    },
    methods: {
      openDialog(data, isEdit){
        this.$nextTick(()=>{
          this.$refs.addDialog.openDialog(data, isEdit);
        })
      },
      searchList(){
        this.$refs.mainTable.search(this.searchParam);
      },
      deleteRecord(data){
        this.$confirm('此操作将永久删除该记录, 是否继续?', '提示', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }).then(() => {
          fetch('/device/delete', {
            method: 'POST',
            body: JSON.stringify(data),
            headers: new Headers({
              'Content-Type': 'application/json'
            })
          }).then(res => {
            return res.json()
          }).then(data => {
            this.searchList();
            this.$message({
              type: 'success',
              message: '删除成功!'
            });
          })
        })
      },
    },
    template: `
      <el-card class="box-card">
      <div slot="header" class="clearfix">
        <span>设备列表</span>
        <el-input v-model="searchParam.id" @keyup.native.enter="searchList" style="width:200px;"></el-input>
        <el-button type="text" @click="searchList">查询</el-button>
        <el-button type="text" @click="openDialog(null, false)">添加</el-button>
      </div>
      <div class="text item">
        <my-table url="/device/list" ref="mainTable">
          <el-table-column prop="id" label="ID" width="140"/>
          <el-table-column prop="sn" label="SN" width="120"/>
          <el-table-column prop="name" label="名称"/>
          <el-table-column prop="provider" label="厂商"/>
          <el-table-column prop="type" label="类型"/>
          <el-table-column prop="model" label="型号"/>
          <el-table-column prop="onlineStatus" label="在线状态">
            <template slot-scope="scope">
              <el-tag :type="scope.row.onlineStatus == 'onLine' ? 'success': 'info'" size="mini">
              {{scope.row.onlineStatus}}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" :width="200" fixed="right">
            <template slot-scope="scope">
              <el-button @click="openDialog(scope.row, true)" type="text" size="small">编辑</el-button>
              <el-button type="text" size="small" @click="deleteRecord(scope.row)">删除</el-button>
              <switch-opt :deviceId="scope.row.id"/>
              <light-opt :deviceId="scope.row.id"/>
              <screenshot :deviceId="scope.row.id"/>
              <led-file-upload :deviceId="scope.row.id"/>
            </template>
          </el-table-column>
        </my-table>
      </div>
      <add-dialog ref="addDialog" @success="searchList()"></add-dialog>
      </el-card>
    ` 
  }
});