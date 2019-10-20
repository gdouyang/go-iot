define({
    props:{
    },
    data: function () {
      return {
        loading: false,
        content: "",
        deviceId: ""
      }
    },
    mounted(){
    },
    methods: {
      open(deviceId){
        this.deviceId = deviceId;
        this.$refs.dialog1.open()
      },
      publish(){
        this.loading = true;
        console.log(this.content)
        // fetch(`/north/control/xixun/v1/${this.deviceId}/screenShot`, {
        //   method: 'POST',
        //   body: "",
        // })
        // .then(res => res.json())
        // .then(data => {
        //   this.loading = false;
        //   if(data.success){
        //   }else{
        //     this.$message({
        //       type: 'error',
        //       message: data.msg
        //     });
        //   }
        // }).catch(()=> this.loading = false)
      },
    },
    template: `
      <my-dialog title="消息发布" ref="dialog1" @confirm="publish" width="80%">
        <my-tinymce v-model="content"/>
      </my-dialog>
    ` 
  });
