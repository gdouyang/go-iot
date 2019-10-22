define({
    props:{
    },
    data: function () {
      return {
        deviceId: "",
		ruleForm:{interval:100,step:1,align:'bottom',direction:'left',num:-1,html:''},
		rules: { 
		interval: [{required: true, message: '步进时间间隔不能为空', trigger: 'blur'}],
		step: [{required: true, message: '步进距离不能为空', trigger: 'blur'},
			{pattern:/^[1-10]$/, message: '步进距离填写1-10', trigger: 'blur'}],
		num: [{ required: true, message: '播放不能为空', trigger: 'blur'}],
		},
      }
    },
    mounted(){
		this.emptyFormData =  JSON.stringify(this.ruleForm)
    },
    methods: {
      open(deviceId){
        this.deviceId = deviceId;
        this.$refs.dialog1.open()
      },
	  publish(){
         fetch(`/north/control/xixun/v1/${this.deviceId}/msgPublish`, {
           method: 'POST',
           body: JSON.stringify(this.ruleForm),
         })
         .then(res => res.json())
         .then(data => {
			this.$message({
               type: data.success ? 'success' : 'error',
               message: data.msg
             });
			this.$refs.dialog1.close();
         })
      },
      clean(){
         fetch(`/north/control/xixun/v1/${this.deviceId}/msgClear`, {
           method: 'POST',
           body: "",
         })
         .then(res => res.json())
         .then(data => {
             this.$message({
               type: data.success ? 'success' : 'error',
               message: data.msg
             });
         })
      },
	handleClose(){
		this.$refs.ruleForm.clearValidate();
		this.ruleForm = JSON.parse(this.emptyFormData);
	}
    },
    template: `
    <my-dialog title="消息发布" ref="dialog1" @close="handleClose" @confirm="publish" width="80%">
    	<el-form :model="ruleForm" :rules="rules" ref="ruleForm" label-width="140px" class="demo-ruleForm">
		 <el-row>
		  <el-col :span="6">
			<el-form-item label="步进间隔时间(ms)" prop="interval">
	          <el-input v-model="ruleForm.interval"></el-input>
	        </el-form-item>
		  </el-col>
          <el-col :span="6">
			<el-form-item label="步进距离(px)" prop="step">
				<el-input-number v-model="ruleForm.step" :min="1" :max="10"></el-input-number>
        	</el-form-item>
		  </el-col>
          <el-col :span="6">
	        <el-form-item label="位置" prop="align">
	          <el-select v-model="ruleForm.align" placeholder="请选择位置">
	            <el-option label="垂直顶端" value="top"></el-option>
	            <el-option label="垂直居中" value="center"></el-option>
	            <el-option label="垂直底部" value="bottom"></el-option>
	          </el-select>
	        </el-form-item>
		  </el-col>
		  <el-col :span="6">
        	<el-form-item label="移动方向" prop="direction">
                <el-select v-model="ruleForm.direction" placeholder="请选择方向">
                  <el-option label="向左移动" value="left"></el-option>
                  <el-option label="向右移动" value="rigth"></el-option>
                </el-select>
        	</el-form-item>
		  </el-col>
		<el-col :span="12">
        	<el-form-item label="播放次数" prop="num" label-width="140px" >
				<el-input-number v-model="ruleForm.num" :min="-1"></el-input-number>
				<el-link type="info">-1为永久，0为停止</el-link>
        	</el-form-item>
		</el-col>
		<el-col :span="12">
			<el-form-item>
	          <el-button @click="clean" type="primary">清除本机上消息</el-button>
	        </el-form-item>
		</el-col>		
        </el-row>
		 <el-row>
			<el-form-item label="消息">
        		<my-tinymce v-model="ruleForm.html"/>
			</el-form-item>
		 </el-row>
		</el-form>
    </my-dialog>
    ` 
  });
