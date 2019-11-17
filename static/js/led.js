define(["components/led_add", "components/switchOpt", "components/lightOpt", "components/onLineStatusOpt", 
"xixunled/ledFileUpload", "xixunled/screenshot","xixunled/ledPlay","xixunled/msgPublish"], 
function(ledAdd, switchOpt, lightOpt, onLineStatusOpt, ledFileUpload, screenshot,ledPlay, msgPublish) {
  return {
    components:{
      'add-dialog': ledAdd,
      'switch-opt': switchOpt,
      'light-opt': lightOpt,
      'onLineStatusOpt': onLineStatusOpt,
      'led-file-upload': ledFileUpload,
      'screenshot': screenshot,
      'ledPlay':ledPlay,
      'msgPublish':msgPublish,
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
      openMsg(data){
        this.$nextTick(()=>{
          this.$refs.msgDialog.open(data.id);
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
          fetch('/north/led/delete', {
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
               type: data.success ? 'success' : 'error',
               message: data.msg
            });
          })
        })
      },
    },
    template: `
      <el-card class="box-card">
      <div slot="header" class="clearfix">
        <span>LED列表</span>
        <el-input v-model="searchParam.id" @keyup.native.enter="searchList" style="width:200px;"></el-input>
        <el-button type="text" @click="searchList">查询</el-button>
        <el-button type="text" @click="openDialog(null, false)">添加</el-button>
      </div>
      <div class="text item">
        <my-table url="/led/list" ref="mainTable">
          <el-table-column prop="id" label="ID"/>
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
          <el-table-column prop="agent" label="Agent"/>
          <el-table-column label="操作" :width="200" fixed="right">
            <template slot-scope="scope">
              <el-button @click="openDialog(scope.row, true)" type="text" size="small">编辑</el-button>
              <el-button type="text" size="small" @click="deleteRecord(scope.row)">删除</el-button>
              <onLineStatusOpt :deviceId="scope.row.id" @success="searchList()"/>
              <switch-opt :deviceId="scope.row.id"/>
              <light-opt :deviceId="scope.row.id"/>
              <screenshot :deviceId="scope.row.id"/>
              <led-file-upload :deviceId="scope.row.id"/>
			        <ledPlay :deviceId="scope.row.id"/>
              <el-button type="text" size="small" @click="openMsg(scope.row)">消息发布</el-button>
            </template>
          </el-table-column>
        </my-table>
      </div>
      <add-dialog ref="addDialog" @success="searchList()"></add-dialog>
      <msgPublish ref="msgDialog"/>
      </el-card>
    ` 
  }
});