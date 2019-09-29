const WebSocket = {
  data: function () {
    return {
      msgList: [{
        content: 'websocket connected!'
      },{
        content: '消息提示的文案1'
      }]
    }
  },
  methods: {
    clear() {
      this.msgList = [];
    }
  },
  template: `
    <el-card class="box-card">
    <div slot="header" class="clearfix">
      <span>WebSocket</span>
      <el-button style="float: right; padding: 3px 0" type="text" @click="clear">清空消息面板</el-button>
    </div>
    <div class="text item">
      <el-alert type="info" effect="dark" style="margin-bottom: 5px;"
        v-for="(msg, index) in msgList" :key="index" :title="msg.content"></el-alert>
    </div>
    </el-card>
  ` 
}