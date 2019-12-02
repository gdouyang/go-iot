package plan

import (
	"database/sql"
	"encoding/json"
	"errors"
	"go-iot/models"

	"github.com/astaxie/beego"
	"github.com/astaxie/beego/toolbox"
)

type Plan struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	Cron      string `json:"cron"`
	Actions   string `json:"actions"`
}

func init() {
	db, _ := getDb()
	defer db.Close()
	_, err := db.Exec(`
		CREATE TABLE plan (
	    id_ INTEGER PRIMARY KEY AUTOINCREMENT,
	    name_ VARCHAR(64) NULL,
		type_ VARCHAR(32) NULL,
		startTime_ VARCHAR(128) NULL,
		endTime_ VARCHAR(32) NULL,
		cron_ VARCHAR(32) NULL,
		actions_ TEXT NULL,
	    created_ DATE NULL
		);
	`)
	if err != nil {
		beego.Info("table Plan create fail:", err)
	} else {
		beego.Info("table Plan create success")
	}
	runPlan()
}

func getDb() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./db/plan.db")
	if err != nil {
		beego.Info("open sqlite fail")
	}
	return db, err
}

func runPlan() {
	//查询数据
	db, _ := getDb()
	defer db.Close()
	sql := "SELECT id_,name_,type_,startTime_,endTime_,cron_ FROM plan "
	rows, err := db.Query(sql)
	if err != nil {
		beego.Error(err)
		return
	}
	var result []Plan
	var (
		Id                                   int
		Name, Type, StartTime, EndTime, Cron string
	)
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&Id, &Name, &Type, &StartTime, &EndTime, &Cron)

		m := Plan{Id: Id, Name: Name, Type: Type, StartTime: StartTime, EndTime: EndTime, Cron: Cron}
		result = append(result, m)
	}

	for _, p := range result {
		AddTask(p)
	}
}

// 分页查询Plan
func ListPlan(page *models.PageQuery) (*models.PageResult, error) {
	var pr *models.PageResult
	var dev Plan
	json.Unmarshal(page.Condition, &dev)

	//查询数据
	db, _ := getDb()
	defer db.Close()
	sql := "SELECT id_,name_,type_,startTime_,endTime_,cron_ FROM plan "
	countSql := "SELECT count(*) from plan"
	name := dev.Name
	params := make([]interface{}, 0)
	if name != "" {
		sql += " where name_ like ?"
		countSql += " where name_ like ?"
		params = append(params, name)
	}
	sql += " limit ? offset ?"
	params = append(params, page.PageSize, page.PageOffset())
	rows, err := db.Query(sql, params...)
	if err != nil {
		return nil, err
	}
	var result []Plan
	var (
		Id                                   int
		Name, Type, StartTime, EndTime, Cron string
	)
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&Id, &Name, &Type, &StartTime, &EndTime, &Cron)

		m := Plan{Id: Id, Name: Name, Type: Type, StartTime: StartTime, EndTime: EndTime, Cron: Cron}
		result = append(result, m)
	}

	rows, err = db.Query(countSql, params...)
	if err != nil {
		return nil, err
	}
	count := 0
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&count)
		break
	}

	pr = &models.PageResult{page.PageSize, page.PageNum, count, result}

	return pr, nil
}

// 添加Plan
func AddPlan(ob *Plan) error {
	rs, err := GetPlanByName(ob.Name)
	if err != nil {
		return err
	}
	if rs.Name != "" {
		return errors.New("Plan已存在!")
	}
	//插入数据
	db, _ := getDb()
	defer db.Close()
	stmt, _ := db.Prepare(`
	INSERT INTO plan(name_, type_, startTime_,endTime_, cron_, actions_, created_) 
	values(?,?,?,?,?,?,datetime('now'))
	`)

	_, err = stmt.Exec(ob.Name, ob.Type, ob.StartTime, ob.EndTime, ob.Cron, ob.Actions)
	if err != nil {
		return err
	}
	AddTask(*ob)

	return nil
}

func DeletePlan(ob *Plan) error {
	db, _ := getDb()
	defer db.Close()
	stmt, _ := db.Prepare("delete from plan where id_=?")

	_, err := stmt.Exec(ob.Id)
	if err != nil {
		return err
	}
	return nil
}

// 更新Plan
func UpdatePlan(ob *Plan) error {
	//更新数据
	db, _ := getDb()
	defer db.Close()
	params := make([]interface{}, 0)

	sql := "update plan set name_=?"
	params = append(params, ob.Name)
	if len(ob.Type) > 0 {
		sql += ", type_ = ?"
		params = append(params, ob.Type)
	}
	if len(ob.StartTime) > 0 {
		sql += ", startTime_ = ?"
		params = append(params, ob.StartTime)
	}
	if len(ob.EndTime) > 0 {
		sql += ", endTime_ = ?"
		params = append(params, ob.EndTime)
	}
	if len(ob.Cron) > 0 {
		sql += ", cron_ = ?"
		params = append(params, ob.Cron)
	}
	sql += " where id_ = ?"
	params = append(params, ob.Id)
	stmt, err := db.Prepare(sql)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(params...)
	if err != nil {
		return err
	}
	AddTask(*ob)
	return nil
}

// 根据name查询Plan
func GetPlanByName(name string) (Plan, error) {
	var result Plan
	db, _ := getDb()
	defer db.Close()
	sql := "SELECT id_,name_,type_ FROM plan where name_ = ?"
	rows, err := db.Query(sql, name)
	if err != nil {
		return result, err
	}
	var (
		Id         int
		Name, Type string
	)
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&Id, &Name, &Type)
		result = Plan{Id: Id, Name: Name, Type: Type}
		break
	}
	return result, nil
}

func AddTask(plan Plan) (err error) {
	defer func() {
		beego.Error("defer caller")
		if e := recover(); e != nil {
			beego.Error(e)
			err = errors.New(e.(string))
		}
	}()
	toolbox.DeleteTask(plan.Name)
	tk := toolbox.NewTask(plan.Name, plan.Cron, func() error {
		beego.Info(plan)
		return nil
	})

	toolbox.StartTask()
	toolbox.AddTask(tk.Taskname, tk)
	return nil
}
