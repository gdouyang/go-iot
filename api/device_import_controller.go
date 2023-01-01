package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"go-iot/models"
	device "go-iot/models/device"
	"log"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/beego/beego/v2/server/web"
	"github.com/xuri/excelize/v2"
)

// 设备管理
func init() {
	ns := web.NewNamespace("/api/device",
		web.NSRouter("/:productId/template", &DeviceImportController{}, "get:Download"),
		web.NSRouter("/:productId/import", &DeviceImportController{}, "post:Import"),
		web.NSRouter("/import-result/:token", &DeviceImportController{}, "get:ImportProcess"),
	)
	web.AddNamespace(ns)

}

type DeviceImportController struct {
	AuthController
}

// 查询设备列表
func (ctl *DeviceImportController) Download() {
	if ctl.isForbidden(deviceResource, QueryAction) {
		return
	}
	productId := ctl.Param(":productId")
	product, err := device.GetProductMust(productId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	if product.CreateId != ctl.GetCurrentUser().Id {
		ctl.RespError(errors.New("product is not you created"))
		return
	}
	xlsx := excelize.NewFile()
	xlsx.SetCellStr("Sheet1", "A1", "deviceId")
	xlsx.SetCellStr("Sheet1", "B1", "name")
	axis := 'C'
	for _, v := range product.Metaconfig {
		pos := fmt.Sprintf("%s1", string(axis))
		xlsx.SetCellStr("Sheet1", pos, v.Property)
		axis = axis + 1
	}
	xlsx.SetActiveSheet(0)
	disposition := fmt.Sprintf("attachment; filename=%s导入模板.xlsx", url.QueryEscape(productId))
	ctl.Ctx.ResponseWriter.Header().Set("Content-Type", "application/octet-stream")
	ctl.Ctx.ResponseWriter.Header().Set("Content-Disposition", disposition)
	ctl.Ctx.ResponseWriter.Header().Set("Content-Transfer-Encoding", "binary")
	ctl.Ctx.ResponseWriter.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")
	xlsx.Write(ctl.Ctx.ResponseWriter)
}

var sseCache map[string]string = make(map[string]string)

// 查询单个设备
func (ctl *DeviceImportController) Import() {
	if ctl.isForbidden(deviceResource, QueryAction) {
		return
	}
	productId := ctl.Param(":productId")
	product, err := device.GetProductMust(productId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	if product.CreateId != ctl.GetCurrentUser().Id {
		ctl.RespError(errors.New("product is not you created"))
		return
	}
	f, _, err := ctl.GetFile("file")
	if err != nil {
		ctl.RespError(err)
		return
	}
	defer f.Close()
	xlsx, err := excelize.OpenReader(f)
	if err != nil {
		ctl.RespError(err)
		return
	}
	sheetname := xlsx.GetSheetName(xlsx.GetActiveSheetIndex())
	rows, err := xlsx.GetRows(sheetname)
	if err != nil {
		ctl.RespError(err)
		return
	}
	var productMetaconfig map[string]bool = make(map[string]bool)
	for _, v := range product.Metaconfig {
		productMetaconfig[v.Property] = true
	}
	var devices []models.Device
	for rowIdx, row := range rows {
		if rowIdx == 0 {
			continue
		}
		dev := models.Device{Id: row[0], Name: row[1]}
		dev.ProductId = productId
		dev.CreateId = ctl.GetCurrentUser().Id
		var devMetaconfig map[string]string = map[string]string{}
		for i := 2; i < len(row); i++ {
			head := rows[0][i]
			if _, ok := productMetaconfig[head]; ok {
				devMetaconfig[head] = row[i]
			}
		}
		str, _ := json.Marshal(devMetaconfig)
		dev.Metaconfig = string(str)
		devices = append(devices, dev)
	}
	token := fmt.Sprintf("%v", time.Now().UnixMicro())
	go func() {
		total := 0
		resp := `{"success":true, "result": {"finish": %v, "num":%d}}`
		for _, data := range devices {
			err := device.AddDevice(&data)
			if err == nil {
				total = total + 1
			}
			if total%5 == 0 {
				{
					mutex := sync.Mutex{}
					mutex.Lock()
					defer mutex.Unlock()
					sseCache[token] = fmt.Sprintf(resp, false, total)
				}
			}
		}
		{
			mutex := sync.Mutex{}
			mutex.Lock()
			defer mutex.Unlock()
			sseCache[token] = fmt.Sprintf(resp, true, total)
		}
	}()
	ctl.RespOkData(token)
}

func (ctl *DeviceImportController) ImportProcess() {
	token := ctl.Param(":token")
	w := ctl.Ctx.ResponseWriter.ResponseWriter
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Panic("server not support")
	}
	for i := 0; i < 6; i++ {
		if len(sseCache[token]) > 0 {
			fmt.Fprintf(w, "data: %s\n\n", sseCache[token])

			{
				mutex := sync.Mutex{}
				mutex.Lock()
				defer mutex.Unlock()
				sseCache[token] = ""
			}

			flusher.Flush()
		}
		time.Sleep(2 * time.Second)
	}
	fmt.Fprintf(w, "event: close\ndata: close\n\n") // 一定要带上data，否则无效
}
