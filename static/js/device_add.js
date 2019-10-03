define({
    data: function () {
      return {
        dialogVisible:false,
        createForm: {id: '', sn: '', name: ''}
      }
    },
    mounted(){
    },
    methods: {
      openDialog(){
        this.dialogVisible = true;
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
              this.$emit('success')
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
    ` 
  });