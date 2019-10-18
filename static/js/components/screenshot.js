define({
    props:{
      deviceId:null
    },
    data: function () {
      return {
        screenshot:null,
        yet: false,
        loading: false
      }
    },
    mounted(){
    },
    methods: {
      screenShot(){
        this.loading = true;
        fetch(`/north/control/xixun/v1/${this.deviceId}/screenShot`, {
          method: 'POST',
          body: "",
        })
        .then(res => res.json())
        .then(data => {
          this.loading = false;
          if(data.success){
            this.yet = true
            this.screenshot = "data:image/png;base64,"+data.msg
          }else{
            this.$message({
              type: 'error',
              message: data.msg
            });
          }
        }).catch(()=> this.loading = false)
      },
    },
    template: `
    <el-popover placement="bottom" width="200" height="150" trigger="click" @show="screenShot" v-loading="loading">
      <el-button type="text" size="small" slot="reference">截图</el-button>
      <el-button type="text" @click="screenShot">开始截图</el-button>
      <div>
        <img v-if="yet" :src="screenshot" style="width: 100%;"/>
      </div>
    </el-popover>
    ` 
  });