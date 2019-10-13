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
      open(command){
        fetch(`/north/control/${this.deviceId}/switch`, {
          method: 'POST',
          body: JSON.stringify([{index:0,status:command}]),
          headers: new Headers({
            'Content-Type': 'application/json'
          })
        }).then(res => {
          return res.json()
        }).then(data => {
          this.$message({
            type: 'success',
            message: data.msg
          });
        })
      },
    },
    template: `
    <el-dropdown @command="open" size="mini">
      <el-button type="text" class="el-dropdown-link">
      开关<i class="el-icon-arrow-down el-icon--right"></i>
      </el-button>
      <el-dropdown-menu slot="dropdown">
        <el-dropdown-item command="open" style="width:50px;">打 开</el-dropdown-item>
        <el-dropdown-item command="close" divided style="width:50px;">关 闭</el-dropdown-item>
      </el-dropdown-menu>
    </el-dropdown>
    ` 
  });