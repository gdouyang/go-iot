define({
    props:{
      value:{required:true}
    },
    watch: {
      value(newVal, oldVal){
        if(!newVal){
          this.labelValue = '';
        }
      }
    },
    data: function () {
      return {
        labelValue: '',
        visible:false,
        agentList:[]
      }
    },
    created(){
      if(this.value){
        this.labelValue = this.value;
      }
    },
    mounted(){
      this.$refs.mainTable.search()
    },
    methods: {
      handlerClick(){
        this.visible = !this.visible;
      },
      loadDong(datas){
        this.agentList = datas
      },
      select(data){
        this.labelValue = data.sn
        this.visible = false;
        this.$emit('input', data.sn)
      }
    },
    template: `
    <el-popover placement="bottom" width="420" v-model="visible">
      <el-input slot="reference" v-model="labelValue" :readonly="true" class="cursor-pointer" @click="handlerClick">
      <i slot="suffix" class="el-input__icon el-icon-arrow-up" 
        :class="{'is-reverse': visible}"></i>
      </el-input>
      <div>
        <my-table url="/agent/list" ref="mainTable" :selectable="false" @done-load="loadDong">
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