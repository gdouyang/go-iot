Vue.component('my-menu', {
	template: `
		<el-menu
	      default-active="2"
	      class="el-menu-vertical-demo">
	      <el-menu-item index="1">
	        <i class="el-icon-menu"></i>
	        <span slot="title"><router-link to="/echows">echo</router-link></span>
	      </el-menu-item>
	      <el-menu-item index="2">
	        <i class="el-icon-document"></i>
	        <span slot="title"><router-link to="/restful">restful</router-link></span>
	      </el-menu-item>
	      <el-menu-item index="3">
	        <i class="el-icon-document"></i>
	        <span slot="title"><router-link to="/device">Device</router-link></span>
		  </el-menu-item>
		  <el-menu-item index="4">
	        <i class="el-icon-document"></i>
	        <span slot="title"><router-link to="/northwebsocket">north websocket</router-link></span>
	      </el-menu-item>
	    </el-menu>
	`
});
