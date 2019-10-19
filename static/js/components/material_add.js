define({
  data: function () {
    return {
      isEdit: false,
      createForm: {id: '', name: '', type:'',file:''},
      providerList: [],
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
    selectFile(){
      let file = this.getSelectFile();
      if(file){
       this.createForm.file = file.name;
      }
    },
    getSelectFile(){
      let file = document.getElementById("file")
      if(file.files.length >0){
        return file.files[0];
      }
      return null
    },
    save(){
      this.$refs.creteForm.validate((valid)=>{
        if (valid) {
          let url = '/material/add';
          if(this.isEdit) {
            url = '/material/update'
          }
          let filedata = new FormData();
          let file = document.getElementById("file")
          if(file.files.length >0){
            filedata.append('uploadname', file.files[0], file.files[0].name);
          }
          filedata.append('name', this.createForm.name);
          filedata.append('type', this.createForm.type);
          filedata.append('id', this.createForm.id);
          let request = new Request(url, {
              method: 'POST',
              credentials: 'include',
              body: filedata,
          });
          fetch(request)
          .then(res => res.json())
          .then(data => {
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
      this.$refs.creteForm.$el.reset();
      this.createForm = JSON.parse(this.emptyFormData);
    },
  },
  template: `
    <my-dialog :title="title" ref="addDialog" @close="handleClose" @confirm="save()">
      <el-form label-position="right" label-width="80px" size="mini" :model="createForm" ref="creteForm">
        <el-row>
          <el-col :span="24">
            <el-form-item label="名称" prop="name" :rules="[{ required: true, message: '不能为空'}]">
              <el-input v-model="createForm.name"></el-input>
            </el-form-item>
          </el-col>
          <!-- <el-col :span="12">
            <el-form-item label="类型" prop="type" :rules="[{ required: true, message: '不能为空'}]">
              <el-select v-model="createForm.type">
                <el-option label="mp4" value="mp4"></el-option>
              </el-select>
            </el-form-item>
          </el-col>
          -->
        </el-row>
        <el-row>
          <el-col :span="24">
            <el-form-item label="型号" prop="file" :rules="[{ required: !isEdit, message: '不能为空'}]">
              <input type="file" id="file" @change="selectFile"/>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </my-dialog>
  ` 
});