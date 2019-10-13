define(["device_add"], function(deviceAdd) {
  return {
    components:{
      'add-dialog': deviceAdd
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
      open(command, data){
        fetch(`/north/control/${data.id}/switch`, {
          method: 'POST',
          body: JSON.stringify([{index:0,status:command}]),
          headers: new Headers({
            'Content-Type': 'application/json'
          })
        }).then(res => {
          return res.json()
        }).then(data => {
          this.$message({
            type: 'success',
            message: data.msg
          });
        })
      },
      status(data){
        fetch('/north/control/status', {
          method: 'POST',
          body: JSON.stringify(data),
          headers: new Headers({
            'Content-Type': 'application/json'
          })
        }).then(res => {
          return res.json()
        }).then(data => {
          this.$message({
            type: 'success',
            message: JSON.stringify(data)
          });
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
          <el-table-column label="操作">
            <template slot-scope="scope">
              <el-button @click="openDialog(scope.row, true)" type="text" size="small">编辑</el-button>
              <el-button type="text" size="small" @click="deleteRecord(scope.row)">删除</el-button>
              <el-dropdown @command="open($event, scope.row)" size="mini">
                <el-button type="text" class="el-dropdown-link">
                开关<i class="el-icon-arrow-down el-icon--right"></i>
                </el-button>
                <el-dropdown-menu slot="dropdown">
                  <el-dropdown-item command="open"> 打开 </el-dropdown-item>
                  <el-dropdown-item command="close" divided> 关闭 </el-dropdown-item>
                </el-dropdown-menu>
              </el-dropdown>
              <el-button type="text" size="small" @click="status(scope.row)">状态</el-button>
            </template>
          </el-table-column>
        </my-table>
      </div>
      <add-dialog ref="addDialog" @success="searchList()"></add-dialog>
      </el-card>
    ` 
  }
});