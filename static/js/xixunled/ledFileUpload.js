define({
    props:{
      deviceId:null
    },
    data: function () {
      return {
        serverUrl: window.location.protocol+"//"+window.location.host
      }
    },
    mounted(){
      this.$refs.table.search()
    },
    methods: {
      upload(){
        let idArray = this.$refs.table.getCheckData("id");
        if(idArray.length < 1){
          this.$message({
            type: 'error',
            message: '请选择需要上传的素材'
          });
          return;
        }
        fetch(`/north/control/xixun/v1/${this.deviceId}/fileUpload`, {
          method: 'POST',
          body: JSON.stringify({
            serverUrl: this.serverUrl,
            ids: idArray.join(',')
          }),
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
    <el-popover placement="bottom" width="400" trigger="click">
      <el-button type="text" size="small" slot="reference">上传</el-button>
      <div>
        <my-table url="/material/list" ref="table" :selectable="true">
          <el-table-column prop="id" label="ID"/>
          <el-table-column prop="name" label="名称"/>
          <el-table-column prop="path" label="路径" width="120" show-overflow-tooltip/>
        </my-table>
      </div>
      <el-input v-model="serverUrl" size="mini" style="width:200px;"/>
      <el-button type="text" @click="upload">提交</el-button>
    </el-popover>
    ` 
  });