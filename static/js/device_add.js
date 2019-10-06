define({
    data: function () {
      return {
        isEdit: false,
        createForm: {id: '', sn: '', name: ''}
      }
    },
    mounted(){
    },
    computed:{
      title(){
        return this.isEdit ? '修改' : '新增';
      }
    },
    methods: {
      openDialog(data, isEdit){
        this.$refs.addDialog.open();
        this.isEdit = isEdit;
        if(data){
          this.createForm = data;
        }
      },
      save(){
        this.$refs.creteForm.validate((valid)=>{
          if (valid) {
            let url = '/device/add';
            if(this.isEdit) {
              url = '/device/update'
            }
            fetch(url, {
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
        this.$refs.creteForm.clearValidate();
        this.createForm = {id: '', sn: '', name: ''};
      }
    },
    template: `
      <my-dialog :title="title" ref="addDialog" @close="handleClose" @confrim="save()">
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
      </my-dialog>
    ` 
  });