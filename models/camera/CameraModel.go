package camera

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"go-iot/models"

	"github.com/astaxie/beego"
	_ "github.com/mattn/go-sqlite3"
)

type Camera struct {
	Id           int    `json:"id"` //ID
	Sn           string `json:"sn"` //SN
	Name         string `json:"name"`
	OnlineStatus string `json:"onlineStatus"` //在线状态
	Host         string `json:"host"`
	RtspPort     int    `json:"rtspPort"`
	OnvifPort    int    `json:"onvifPort"`
	Provider     string `json:"provider"`
	Model        string `json:"model"`
	AuthUser     string `json:"authUser"`
	AuthPass     string `json:"authPass"`
	OnvifUser    string `json:"onvifUser"`
	OnvifPass    string `json:"onvifPass"`
}

func init() {
	db, _ := getDb()
	defer db.Close()
	_, err := db.Exec(`
		CREATE TABLE camera (
	    id_ INTEGER PRIMARY KEY AUTOINCREMENT,
	    sn_ VARCHAR(64) NULL,
		name_ VARCHAR(64) NULL,
		host_ VARCHAR(32) NULL,
		rtsp_port_ INTEGER NULL,
		onvif_port_ INTEGER NULL,
		provider_ VARCHAR(32) NULL,
		model_ VARCHAR(32) NULL,
		auth_user VARCHAR(32) NULL,
		auth_pass_ VARCHAR(256) NULL,
		onvif_user_ VARCHAR(32) NULL,
		onvif_pass_ VARCHAR(256) NULL,
		online_status_ VARCHAR(10) NULL,
	    created_ DATE NULL
		);
	`)
	if err != nil {
		beego.Info("table camera create fail:", err)
	} else {
		beego.Info("table camera create success")
	}
}

func getDb() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./db/camera.db")
	if err != nil {
		beego.Info("open sqlite fail")
	}
	return db, err
}

// 分页查询
func ListCamera(page *models.PageQuery) (*models.PageResult, error) {
	var dev Camera
	json.Unmarshal(page.Condition, &dev)

	//查询数据
	db, _ := getDb()
	defer db.Close()
	sql := "SELECT id_,sn_,name_,online_status_ FROM camera "
	countSql := "SELECT count(*) from camera"
	sn := dev.Sn
	params := make([]interface{}, 0)
	if sn != "" {
		sql += ` where sn_ like ?`
		countSql += ` where sn_ like ?`
		params = append(params, fmt.Sprintf("%%%s%%", sn))
	}
	sql += " limit ? offset ?"
	params = append(params, page.PageSize, page.PageOffset())
	rows, err := db.Query(sql, params...)
	if err != nil {
		return nil, err
	}
	var result []*Camera
	defer rows.Close()
	for rows.Next() {
		var c = new(Camera)
		rows.Scan(&c.Id, &c.Sn, &c.Name, &c.OnlineStatus)
		result = append(result, c)
	}

	rows, err = db.Query(countSql, params...)
	if err != nil {
		return nil, err
	}
	count := 0
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&count)
	}

	pr := models.PageResult{PageSize: page.PageSize, PageNum: page.PageNum, Total: count, List: result}
	return &pr, nil
}

func AddCamera(ob *Camera) error {
	var err error
	//插入数据
	db, _ := getDb()
	defer db.Close()
	stmt, _ := db.Prepare(`
		insert into camera (
	    sn_ ,
		name_ ,
		host_ ,
		rtsp_port_ ,
		onvif_port_ ,
		provider_ ,
		model_,
		auth_user ,
		auth_pass_ ,
		onvif_user_ ,
		onvif_pass_ ,
		online_status_ 
		) 
		values (?,?,?,?,?,?,?,?,?,?,?,?)
	`)

	if len(ob.OnlineStatus) == 0 {
		ob.OnlineStatus = models.OFFLINE
	}

	_, err = stmt.Exec(ob.Sn, ob.Name, ob.Host, ob.RtspPort, ob.OnvifPort, ob.Provider, ob.Model, ob.AuthUser, ob.AuthPass, ob.OnvifUser, ob.OnvifPass, ob.OnlineStatus)
	if err != nil {
		return err
	}

	return nil
}

func UpdateCamera(id string, ob *Camera) error {
	//更新数据
	db, _ := getDb()
	defer db.Close()
	stmt, err := db.Prepare(`
		update camera set
			sn_ =?,
			name_ =?,
			host_ =?,
			rtsp_port_ =?,
			onvif_port_ =?,
			provider_ =?,
			model_=?,
			auth_user =?,
			auth_pass_ =?,
			onvif_user_ =?,
			onvif_pass_ =?
 		where id_=?
	`)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(ob.Sn, ob.Name, ob.Host, ob.RtspPort, ob.OnvifPort, ob.Provider, ob.Model, ob.AuthUser, ob.AuthPass, ob.OnvifUser, ob.OnvifPass, id)
	if err != nil {
		beego.Error("update fail", err)
		return err
	}
	return nil
}

// 根据SN与provider来更新设备在线状态
func UpdateOnlineStatus(onlineStatus string, sn string) error {
	var ob Camera = Camera{Sn: sn, Name: sn, OnlineStatus: onlineStatus}
	err := AddCamera(&ob)
	if err != nil {
		//更新数据
		db, _ := getDb()
		defer db.Close()
		stmt, err := db.Prepare("update camera set online_status_ = ? where sn_ = ?")
		if err != nil {
			return err
		}

		_, err = stmt.Exec(onlineStatus, sn)
		if err != nil {
			beego.Error("update fail", err)
			return err
		}
	}
	return nil
}

func DeleteCamera(ob *Camera) error {
	db, _ := getDb()
	defer db.Close()
	stmt, _ := db.Prepare("delete from camera where id_=?")

	_, err := stmt.Exec(ob.Id)
	if err != nil {
		beego.Error("delete fail", err)
		return err
	}
	return nil
}

func GetCamera(sn string) (Camera, error) {
	var result Camera
	db, _ := getDb()
	defer db.Close()
	sql := `select
		    sn_ ,
			name_ ,
			host_ ,
			rtsp_port_ ,
			onvif_port_ ,
			provider_ ,
			auth_user ,
			auth_pass_ ,
			onvif_user_ ,
			onvif_pass_ 
			from camera where sn_ = ? `
	rows, err := db.Query(sql, sn)
	if err != nil {
		return result, err
	}
	var (
		Id, OnvifPort, RtspPort                       int
		Sn, Name, OnlineStatus, Host, Provider, Model string
		AuthUser, AuthPass, OnvifUser, OnvifPass      string
	)
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&Id, &Sn, &Name, &Host, &RtspPort, &OnvifPort, &Provider, &Model, &AuthUser, &AuthPass, &OnvifUser, &OnvifPass, &OnlineStatus)
		result = Camera{Id: Id, Sn: Sn, Name: Name, OnlineStatus: OnlineStatus, Host: Host, RtspPort: RtspPort, OnvifPort: OnvifPort, Provider: Provider, Model: Model, AuthUser: AuthUser, AuthPass: AuthPass, OnvifUser: OnvifUser, OnvifPass: OnvifPass}
		break
	}
	return result, nil
}

func GetCameraByID(id int) (*Camera, error) {
	var result = new(Camera)
	db, _ := getDb()
	defer db.Close()
	sql := `select
		    sn_ ,
			name_ ,
			host_ ,
			rtsp_port_ ,
			onvif_port_ ,
			provider_ ,
			model_,
			auth_user ,
			auth_pass_ ,
			onvif_user_ ,
			onvif_pass_ 
			from camera where id_ = ? `
	rows, err := db.Query(sql, id)
	if err != nil {
		return result, err
	}

	defer rows.Close()
	for rows.Next() {
		rows.Scan(&result.Sn, &result.Name, &result.Host, &result.RtspPort, &result.OnvifPort, &result.Provider, &result.Model, &result.AuthUser, &result.AuthPass, &result.OnvifUser, &result.OnvifPass)
		break
	}
	return result, nil
}
