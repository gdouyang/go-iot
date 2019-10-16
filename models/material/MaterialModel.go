package material

// 素材Model给LED播放
import (
	"database/sql"
	"encoding/json"
	"errors"
	"go-iot/models"
	"os"

	"github.com/astaxie/beego"
	_ "github.com/mattn/go-sqlite3"
)

// 素材
type Material struct {
	Id   string `json:"id"` //设备ID
	Name string `json:"name"`
	Type string `json:"type"`
	Path string `json:"path"`
}

func init() {
	db, _ := getDb()
	defer db.Close()
	_, err := db.Exec(`
		CREATE TABLE material (
	    id_ INTEGER PRIMARY KEY AUTOINCREMENT,
	    name_ VARCHAR(64) NULL,
		type_ VARCHAR(32) NULL,
		path_ VARCHAR(128) NULL,
	    created_ DATE NULL
		);
	`)
	if err != nil {
		beego.Info("table material create fail:", err)
	} else {
		beego.Info("table material create success")
	}
}

func getDb() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./db/material.db")
	if err != nil {
		beego.Info("open sqlite fail")
	}
	return db, err
}

// 分页查询素材
func ListMaterial(page *models.PageQuery) (*models.PageResult, error) {
	var pr *models.PageResult
	var dev Material
	json.Unmarshal(page.Condition, &dev)

	//查询数据
	db, _ := getDb()
	defer db.Close()
	sql := "SELECT id_,name_,type_,path_ FROM material "
	countSql := "SELECT count(*) from material"
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
	var result []Material
	var (
		Id, Name, Type, Path string
	)
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&Id, &Name, &Type, &Path)

		m := Material{Id: Id, Name: Name, Type: Type, Path: Path}
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

// 添加素材
func AddMaterial(ob *Material) error {
	rs, err := GetMaterialByName(ob.Name)
	if err != nil {
		return err
	}
	if rs.Id != "" {
		return errors.New("素材已存在!")
	}
	//插入数据
	db, _ := getDb()
	defer db.Close()
	stmt, _ := db.Prepare(`
	INSERT INTO material(name_, type_, path_, created_) 
	values(?,?,?,?)
	`)

	_, err = stmt.Exec(ob.Name, ob.Type, ob.Path, "")
	if err != nil {
		return err
	}

	return nil
}

// 更新素材
func UpdateMaterial(ob *Material) error {
	//更新数据
	db, _ := getDb()
	defer db.Close()
	params := make([]interface{}, 0)

	sql := "update material set name_=?"
	params = append(params, ob.Name)
	if len(ob.Type) > 0 {
		sql += ", type_ = ?"
		params = append(params, ob.Type)
	}
	if len(ob.Path) > 0 {
		sql += ", path_ = ?"
		params = append(params, ob.Path)
	}
	sql += " where id_ = ?"
	params = append(params, ob.Id)
	stmt, err := db.Prepare(sql)
	if err != nil {
		return err
	}

	_, err = stmt.Exec(params...)
	if err != nil {
		beego.Error("update fail", err)
		return err
	}
	return nil
}

// 删除素材
func DeleteMaterial(ob *Material) error {
	//更新数据
	db, _ := getDb()
	defer db.Close()

	m, err := GetMaterialByName(ob.Name)
	if err != nil {
		return err
	}
	// 删除文件
	os.Remove("." + m.Path)
	stmt, err := db.Prepare("delete from material where id_=?")

	_, err = stmt.Exec(ob.Id)
	if err != nil {
		beego.Error("delete fail", err)
		return err
	}
	return nil
}

// 根据name查询素材
func GetMaterialByName(name string) (Material, error) {
	var result Material
	db, _ := getDb()
	defer db.Close()
	sql := "SELECT id_,name_,type_,path_ FROM material where name_ = ?"
	rows, err := db.Query(sql, name)
	if err != nil {
		return result, err
	}
	var (
		Id, Name, Type, Path string
	)
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&Id, &Name, &Type, &Path)
		result = Material{Id: Id, Name: Name, Type: Type, Path: Path}
		break
	}
	return result, nil
}
