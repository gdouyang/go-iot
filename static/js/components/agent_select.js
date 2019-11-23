define({
    props:{
      value:{required:true},
      clearable:{default:true}
    },
    watch: {
      value(newVal, oldVal){
        if(!newVal){
          this.labelValue = '';
        }else{
          this.labelValue = newVal;
        }
      }
    },
    directives:{
      Clickoutside: ELEMENT.Select.directives.Clickoutside
    },
    data: function () {
      return {
        labelValue: '',
        visible:false,
        agentList:[],
        showArrow: true,
        showClose: false
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
      },
      iconSwitch(){
        if(this.clearable && this.value){
          this.showArrow = !this.showArrow;
          this.showClose = !this.showClose;
        }
      },
      clear(){
        this.labelValue = ''
        this.$emit('input', '')
        this.iconSwitch()
      }
    },
    template: `
    <el-popover placement="bottom" width="420" v-model="visible" v-clickoutside="handlerClick">
      <el-input slot="reference" v-model="labelValue" :readonly="true" class="cursor-pointer" @click="handlerClick" 
      @mouseenter.native="iconSwitch" @mouseleave.native="iconSwitch">
      <span slot="suffix">
        <i class="el-input__icon el-icon-arrow-up" :class="{'is-reverse': visible}" v-show="showArrow"></i>
        <i class="el-input__icon el-icon-circle-close cursor-pointer" v-if="showClose" @click.self.stop="clear"></i>
      </span>
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