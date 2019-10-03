define(["device_add"], function(deviceAdd) {
  return {
    components:{
      'add-dialog': deviceAdd
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
      openDialog(){
        this.$nextTick(()=>{
          this.$refs.addDialog.openDialog()
        })
      },
      searchList(){
        fetch('/device/list', {
          method: 'POST', // or 'PUT'
          body: JSON.stringify({}), // data can be `string` or {object}!
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
          this.tableData = data;
        })
      },
    },
    template: `
      <el-card class="box-card">
      <div slot="header" class="clearfix">
        <span>设备列表</span>
        <el-button style="float: right; padding: 3px 0" type="text" @click="openDialog">添加</el-button>
      </div>
      <div class="text item">
        <el-table :data="tableData.list">
          <el-table-column prop="id" label="ID" width="140">
          </el-table-column>
          <el-table-column prop="sn" label="SN" width="120">
          </el-table-column>
          <el-table-column prop="name" label="名称">
          </el-table-column>
        </el-table>
      </div>
      <add-dialog ref="addDialog" @success="searchList()"></add-dialog>
      </el-card>
    ` 
  }
});