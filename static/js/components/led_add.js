define(['components/agent_select'], function(agentSelect){
  return {
    components:{
      'agentSelect':agentSelect
    },
    data: function () {
      return {
        isEdit: false,
        createForm: {id: '', sn: '', name: '',provider:'',type:'led',model:''},
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
        this.getAllProvider();
        if(data){
          this.createForm = data;
        }
      },
      save(){
        this.$refs.creteForm.validate((valid)=>{
          if (valid) {
            let url = '/north/led/add';
            if(this.isEdit) {
              url = '/north/led/update'
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
      },
      getAllProvider(){
        fetch("/led/listProvider", {
          method: 'POST', // or 'PUT'
          body: "",
          headers: new Headers({
            'Content-Type': 'application/json'
          })
        }).then(res => {
          return res.json()
        }).then(data => {
          this.providerList = data;
        })
      }
    },
    template: `
      <my-dialog :title="title" ref="addDialog" @close="handleClose" @confirm="save()">
        <el-form label-position="right" label-width="80px" size="mini" :model="createForm" ref="creteForm">
          <el-row>
            <el-col :span="12">
              <el-form-item label="ID" prop="id" :rules="[{ required: true, message: '不能为空'}]">
                <el-input v-model="createForm.id" :disabled="isEdit" size="mini"></el-input>
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="SN" prop="sn" :rules="[{ required: true, message: '不能为空'}]">
                <el-input v-model="createForm.sn" size="mini"></el-input>
              </el-form-item>
            </el-col>
          </el-row>
          <el-row>
            <el-col :span="12">
              <el-form-item label="名称" prop="name" :rules="[{ required: true, message: '不能为空'}]">
                <el-input v-model="createForm.name" size="mini"></el-input>
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="厂商" prop="provider" :rules="[{ required: true, message: '不能为空'}]">
                <el-select v-model="createForm.provider" size="mini">
                  <el-option v-for="(item, index) in providerList" :key="index"
                    :label="item" :value="item"></el-option>
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>
          <el-row>
            <el-col :span="12">
              <el-form-item label="型号" prop="model" :rules="[{ required: true, message: '不能为空'}]">
                <el-select v-model="createForm.model" size="mini">
                  <el-option label="0001" value="0001"></el-option>
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="Agent" prop="agent">
                <agentSelect v-model="createForm.agent"/>
              </el-form-item>
            </el-col>
          </el-row>
        </el-form>
      </my-dialog>
    ` 
  }
});