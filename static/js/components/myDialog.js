Vue.component('my-dialog', {
    props:{
        title: ""
    },
    data(){
        return {
          dialogVisible: false
        }
    },
    methods:{
        open(callback){
          this.dialogVisible = true;
          if(callback){
            this.$nextTick(()=>{
              callback();
            })
          }
        },
        close(callback){
          this.dialogVisible = false;
          if(callback){
            this.$nextTick(()=>{
              callback();
            })
          }
        },
        confrim(){
          this.$emit('confirm');
        },
        onClose(){
          this.$emit('close');
        }
    },
	template: `
    <div>
      <el-dialog :title="title" :visible.sync="dialogVisible" @close="onClose"
        :close-on-click-modal="false">
        <slot></slot>
        <span slot="footer" class="dialog-footer">
          <el-button @click="close()">取 消</el-button>
          <el-button type="primary" @click="confrim()">确 定</el-button>
        </span>
      </el-dialog>
    </div>
	`
});
