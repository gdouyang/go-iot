define({
    props:{
      id: null
    },
    watch: {
    },
    data: function () {
      return {
        agentList:[]
      }
    },
    created(){
    },
    mounted(){
      this.$refs.mainTable.search()
    },
    methods: {
      loadDong(datas){
        this.agentList = datas
      },
      select(data){
        fetch(`/material/sendToAgent/${this.id}/${data.sn}`, {
          method: 'POST',
          body: '',
          headers: new Headers({
            'Content-Type': 'application/json'
          })
        }).then(res => {
          return res.json()
        }).then(data => {
          this.$message({
            type: data.success ? 'success' : 'error',
            message: data.msg ? data.msg : '发送成功!'
          });
        })
      },
    },
    template: `
    <el-popover placement="bottom" width="420" trigger="click">
      <el-button type="text" size="small" slot="reference">同步到Agent</el-button>
      <div>
        <my-table url="/agent/list" ref="mainTable" :selectable="false">
          <el-table-column prop="id" label="ID" width="50"/>
          <el-table-column prop="sn" label="sn" width="150"/>
          <el-table-column prop="name" label="名称" width="150"/>
          <el-table-column label="操作" width="70">
            <template slot-scope="scope">
              <el-button @click="select(scope.row)" type="text" size="small">选择</el-button>
            </template>
          </el-table-column>
        </my-table>
      </div>
    </el-popover>
    ` 
  });