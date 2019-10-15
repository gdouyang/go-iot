// contents of main.js:
require.config({
    paths: {
        device: 'device',
        restful: 'restful',
        echows: 'websoket',
        material: 'material',
    }
});

Vue.component('my-menu', {
	method:{
		toPage(path){
			this.$router.push({path: path})
		}
	},
	template: `
		<el-menu default-active="2" class="el-menu-vertical-demo" :router="true">
      <el-menu-item index="echows">
        <i class="el-icon-connection"></i>
        <span slot="title">Push Echo</span>
      </el-menu-item>
      <el-menu-item index="restful">
        <i class="el-icon-monitor"></i>
        <span slot="title">Restful</span>
      </el-menu-item>
      <el-menu-item index="device">
        <i class="el-icon-mobile-phone"></i>
        <span slot="title">设备</span>
      </el-menu-item>
      <el-menu-item index="material">
        <i class="el-icon-mobile-phone"></i>
        <span slot="title">素材</span>
      </el-menu-item>
    </el-menu>
	`
});
