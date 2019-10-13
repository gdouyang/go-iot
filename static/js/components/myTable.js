Vue.component('my-table', {
    props:{
        url: ""
    },
    data(){
        return {
            tableData: [],
            page:{
                pageNum: 1,
                pageSize: 10,
                condition: {}
            },
            total: 0
        }
    },
    methods:{
        search(param){
          if(param) {
              this.page.condition = param;
          }
          fetch(this.url, {
            method: 'POST',
            body: JSON.stringify(this.page),
            headers: new Headers({
              'Content-Type': 'application/json'
            })
          }).then(res => {
            return res.json()
          }).then(data => {
            console.log(data)
            if(data.list == null){
              data.list = []
            }
            this.tableData = data.list;
            this.total = data.total;
          })
        }
    },
	template: `
        <div>
        <el-table :data="tableData" style="width: 100%">
            <slot></slot>
        </el-table>
        <el-pagination layout="prev, pager, next" :total="total" 
          :current-page.sync="page.pageNum" :page-size="page.pageSize" @current-change="search()">
        </el-pagination>
        </div>
	`
});
