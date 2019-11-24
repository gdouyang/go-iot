package plan

import (
	planmodel "go-iot/models/plan"

	"github.com/astaxie/beego"

	"github.com/astaxie/beego/toolbox"
)

func AddTask(plan planmodel.Plan) {
	toolbox.DeleteTask(plan.Name)
	tk := toolbox.NewTask(plan.Name, plan.Cron, func() error {
		beego.Info(plan)
		return nil
	})

	toolbox.StartTask()
	toolbox.AddTask(tk.Taskname, tk)
}
