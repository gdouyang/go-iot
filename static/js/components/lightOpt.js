define({
    props:{
      deviceId:null
    },
    data: function () {
      return {
        lightvalue:0
      }
    },
    mounted(){
    },
    methods: {
      light(value){
        fetch(`/north/control/${this.deviceId}/light`, {
          method: 'POST',
          body: JSON.stringify({value:value}),
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
          // this.lightvalue = 0;
        })
      },
    },
    template: `
    <el-dropdown size="mini" :hide-on-click="false">
      <el-button type="text" class="el-dropdown-link">
      调光<i class="el-icon-arrow-down el-icon--right"></i>
      </el-button>
      <el-dropdown-menu slot="dropdown">
        <el-dropdown-item>
        <div style="width:200px; padding:5px;">
        <el-slider v-model="lightvalue" :min="0" :max="100" @change="light(lightvalue)"></el-slider>
        </div>
        </el-dropdown-item>
      </el-dropdown-menu>
    </el-dropdown>
    ` 
  });