const Device = {
  data: function () {
    return {
      tableData: [],
      dialogVisible:false,
      createForm: {id: '', sn: '', name: ''}
    }
  },
  mounted(){
    this.searchList();
  },
  methods: {
    openDialog(){
      this.dialogVisible = true;
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
    save(){
      this.$refs.creteForm.validate((valid)=>{
        if (valid) {
          fetch('/device/add', {
            method: 'POST', // or 'PUT'
            body: JSON.stringify(this.createForm),
            headers: new Headers({
              'Content-Type': 'application/json'
            })
          }).then(res => {
            return res.json()
          }).then(data => {
            this.searchList();
            this.dialogVisible = false;
          })
        }
      })
    },
    handleClose(){
      this.createForm = {id: '', sn: '', name: ''};
    }
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
    <el-dialog title="新增" :visible.sync="dialogVisible" :close="handleClose">
      <el-form label-position="right" label-width="80px" :model="createForm" ref="creteForm">
        <el-form-item label="ID" prop="id" :rules="[{ required: true, message: '不能为空'}]">
          <el-input v-model="createForm.id"></el-input>
        </el-form-item>
        <el-form-item label="SN" prop="sn" :rules="[{ required: true, message: '不能为空'}]">
          <el-input v-model="createForm.sn"></el-input>
        </el-form-item>
        <el-form-item label="名称" prop="name" :rules="[{ required: true, message: '不能为空'}]">
          <el-input v-model="createForm.name"></el-input>
        </el-form-item>
      </el-form>
      <span slot="footer" class="dialog-footer">
        <el-button @click="dialogVisible = false">取 消</el-button>
        <el-button type="primary" @click="save">确 定</el-button>
      </span>
    </el-dialog>
    </el-card>
  ` 
}