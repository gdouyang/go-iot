define(['components/agent_select'], function(agentSelect){
  return {
    components:{
      'agentSelect':agentSelect
    },
    data: function () {
      return {
        isEdit: false,
        id:0,
        createForm: {sn: '', name: '',provider:'',type:'camera',model:'',host:'',rtspPort:'',onvifPort:'',model:'',authUser:'',authPass:'',onvifUser:'',onvifPass:''},
        providerList: ["HIKVISION","UNV","HUAWEI"],
		modelList: ["IPC","NVR4300","IVS","5800"],
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
      openDialog(data, isEdit,id){
        this.$refs.addDialog.open();
        this.isEdit = isEdit;
        this.id = id
        if(data){
          this.createForm = data;
        }
      },
      save(){
        this.$refs.createForm.validate((valid)=>{
          if (valid) {
            let url = '/camera/add';
            if(this.isEdit) {
              url = '/camera/update/' + this.id
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
        this.$refs.createForm.clearValidate();
        this.createForm = JSON.parse(this.emptyFormData);
      }
    },
    template: `
      <my-dialog :title="title" ref="addDialog" @close="handleClose" @confirm="save()">
        <el-form label-position="right" label-width="80px" size="mini" :model="createForm" ref="createForm">
          <el-row>
            <el-col :span="12">
              <el-form-item label="SN" prop="sn" :rules="[{ required: true, message: '不能为空'}]">
                <el-input v-model="createForm.sn"></el-input>
              </el-form-item>
            </el-col>
			<el-col :span="12">
              <el-form-item label="Agent" prop="agent">
                <agentSelect v-model="createForm.agent"/>
              </el-form-item>
            </el-col>
          </el-row>
          <el-row>
            <el-col :span="12">
              <el-form-item label="名称" prop="name" :rules="[{ required: true, message: '不能为空'}]">
                <el-input v-model="createForm.name"></el-input>
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="厂商" prop="provider" :rules="[{ required: true, message: '不能为空'}]">
                <el-select v-model="createForm.provider">
                  <el-option v-for="(item, index) in providerList" :key="index"
                    :label="item" :value="item"></el-option>
                </el-select>
              </el-form-item>
            </el-col>
          </el-row>
          <el-row>
          <el-col :span="12">
			          <el-form-item label="型号" prop="model" :rules="[{ required: true, message: '不能为空'}]">
                <el-select v-model="createForm.model">
                  <el-option v-for="(item, index) in modelList" :key="index"
                    :label="item" :value="item"></el-option>
                </el-select>
                </el-form-item>
          </el-col>
			    <el-col :span="12">
			          <el-form-item label="IP地址" prop="host" :rules="[{required: true, message: '不能为空'}]">
                <el-input v-model="createForm.host"></el-input>
              </el-form-item>
          </el-col>
          </el-row>
          <el-row>
          <el-col :span="12">
			          <el-form-item label="RTST端口" prop="rtspPort">
                <el-input-number :min="1" :max="65535" v-model="createForm.rtspPort"></el-input>
              </el-form-item>
          </el-col>
          <el-col :span="12">
			        <el-form-item label="用户" prop="authUser" :rules="[{ required: true, message: '不能为空'}]">
                <el-input v-model="createForm.authUser"></el-input>
              </el-form-item>
            </el-col>
          </el-row> 
          <el-row>
          <el-col :span="12">
			<el-form-item label="密码" prop="authPass" :rules="[{ required: true, message: '不能为空'}]">
                <el-input v-model="createForm.authPass"></el-input>
              </el-form-item>
			</el-col>
			<el-col :span="12">
			<el-form-item label="ONVIF端口" prop="onvifPort">
                <el-input-number :min="1" :max="65535" v-model="createForm.onvifPort"></el-input>
              </el-form-item>
			</el-col>
      </el-row>
      <el-row>
          <el-col :span="12">
			<el-form-item label="ONVIF用户" prop="onvifUser" :rules="[{ required: true, message: '不能为空'}]">
                <el-input v-model="createForm.onvifUser"></el-input>
              </el-form-item>
              </el-col>
              <el-col :span="12">
			<el-form-item label="ONVIF密码" prop="onvifPass" :rules="[{ required: true, message: '不能为空'}]">
                <el-input v-model="createForm.onvifPass"></el-input>
              </el-form-item>
              </el-col>
      </el-row>
        </el-form>
      </my-dialog>
    ` 
  }
});