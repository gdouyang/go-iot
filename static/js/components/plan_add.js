define({
  data: function () {
    return {
      isEdit: false,
      createForm: {id: '', name: '', type:'',startTime:'',endTime:'',cron:'', actions:""},
      actions:[{action:'',deviceIds:[]}],
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
          let url = '/north/plan/add';
          if(this.isEdit) {
            url = '/north/plan/update'
          }
          this.createForm.actions = JSON.stringify(this.actions);
          let request = new Request(url, {
              method: 'POST',
              credentials: 'include',
              body: JSON.stringify(this.createForm),
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
          <el-col :span="12">
            <el-form-item label="名称" prop="name" :rules="[{ required: true, message: '不能为空'}]">
              <el-input v-model="createForm.name"></el-input>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="类型" prop="type" :rules="[{ required: true, message: '不能为空'}]">
              <el-select v-model="createForm.type">
                <el-option label="iot" value="Iot"></el-option>
                <el-option label="agent" value="Agent"></el-option>
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="12">
            <el-form-item label="开始时间" prop="startTime" :rules="[{ required: true, message: '不能为空'}]">
              <el-input v-model="createForm.startTime"></el-input>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="结束时间" prop="endTime" :rules="[{ required: true, message: '不能为空'}]">
              <el-input v-model="createForm.endTime"></el-input>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row>
          <el-col :span="12">
            <el-form-item label="cron" prop="cron" :rules="[{ required: true, message: '不能为空'}]">
              <el-input v-model="createForm.cron"></el-input>
            </el-form-item>
          </el-col>
        </el-row>
        <el-divider content-position="left">动作</el-divider>
        <el-row v-for="(item,index) in actions" :key="index">
          <el-col :span="12">
            <el-form-item label="动作">
              <el-select v-model="item.action">
                <el-option label="开" value="open"></el-option>
                <el-option label="关" value="close"></el-option>
                <el-option label="调光" value="light"></el-option>
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="设备">
              <el-select v-model="item.deviceIds" multiple>
                <el-option label="ld0002" value="ld0002"></el-option>
                <el-option label="ld0003" value="ld0003"></el-option>
                <el-option label="ld0004" value="ld0004"></el-option>
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </my-dialog>
  ` 
});