define(["camera/camera_add"], 
function(cameraAdd) {
  return {
    components:{
      'add-dialog': cameraAdd,
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
      getDetail(abc){
		let cameraid = abc.id;
        fetch('/camera/detail/'+cameraid, {
          method: 'GET'
        }).then(res => {
          return res.json()
        }).then(data => {
          if (!data.success && "undefined" != typeof data.success){
            this.searchList();
            this.$message({
              type: 'false',
              message: '查找的设备已经不存在了'
            });
          } else {
            this.openDialog(data,true);
          }
        })
      },
      deleteRecord(data){
        this.$confirm('此操作将永久删除该记录, 是否继续?', '提示', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }).then(() => {
          fetch('/camera/delete', {
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
        <span>ID</span>
        <el-input v-model="searchParam.id" @keyup.native.enter="searchList" style="width:200px;"></el-input>
        <el-button type="text" @click="searchList">查询</el-button>
        <el-button type="text" @click="openDialog(null, false)">添加</el-button>
      </div>
      <div class="text item">
        <my-table url="/camera/list" ref="mainTable">
          <el-table-column prop="id" label="ID" width="70"/>
          <el-table-column prop="sn" label="SN" min-width="120"/>
          <el-table-column prop="name" label="名称"/>
          <el-table-column prop="onlineStatus" label="在线状态">
            <template slot-scope="scope">
              <el-tag :type="scope.row.onlineStatus == 'onLine' ? 'success': 'info'" size="mini">
              {{scope.row.onlineStatus}}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" :width="200" fixed="right">
            <template slot-scope="scope">
              <el-button @click="getDetail(scope.row)" type="text" size="small">编辑</el-button>
              <el-button type="text" size="small" @click="deleteRecord(scope.row)">删除</el-button>
            </template>
          </el-table-column>
        </my-table>
      </div>
      <add-dialog ref="addDialog" @success="searchList()"></add-dialog>
      </el-card>
    ` 
  }
});