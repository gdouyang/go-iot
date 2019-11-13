define({
    props:{
      deviceId:null
    },
    data: function () {
      return {
      }
    },
    mounted(){
    },
    methods: {
      getOnlineStatus(command){
        fetch(`/north/control/${this.deviceId}/get/online-status`, {
          method: 'POST',
          body: "",
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
        })
      },
    },
    template: `
      <el-button type="text" size="small" @click="getOnlineStatus">
      刷新状态
      </el-button>
    </el-dropdown>
    ` 
  });