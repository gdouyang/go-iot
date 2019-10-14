define({
    data: function () {
      return {
        isEdit: false,
        createForm: {id: '', sn: '', name: '',provider:'',type:'',model:''}
      }
    },
    mounted(){
      this.emptyFormData = JSON.stringify(this.createForm)
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
              this.$message({
                type: data.success ? 'success' : 'error',
                message: data.msg
              });
              if(data.success){
                this.$emit('success')
                this.$refs.addDialog.close();
              }
            })
          }
        })
      },
      handleClose(){
        this.$refs.creteForm.clearValidate();
        this.createForm = JSON.parse(this.emptyFormData);
      }
    },
    template: `
      <my-dialog :title="title" ref="addDialog" @close="handleClose" @confirm="save()">
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
          <el-form-item label="厂商" prop="provider" :rules="[{ required: true, message: '不能为空'}]">
            <el-input v-model="createForm.provider"></el-input>
          </el-form-item>
          <el-form-item label="类型" prop="type" :rules="[{ required: true, message: '不能为空'}]">
            <el-select v-model="createForm.type">
              <el-option label="LED" value="led"></el-option>
            </el-select>
          </el-form-item>
          <el-form-item label="型号" prop="model" :rules="[{ required: true, message: '不能为空'}]">
            <el-select v-model="createForm.model">
              <el-option label="0001" value="0001"></el-option>
            </el-select>
          </el-form-item>
        </el-form>
      </my-dialog>
    ` 
  });