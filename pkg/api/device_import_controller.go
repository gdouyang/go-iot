package api

import (
	"errors"
	"fmt"
	"go-iot/pkg/api/web"
	"go-iot/pkg/models"
	device "go-iot/pkg/models/device"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	logs "go-iot/pkg/logger"

	"github.com/xuri/excelize/v2"
)

// 设备导入
func init() {
	var d = &deviceApi{}
	// 导出模板
	web.RegisterAPI("/device/{productId}/template", "GET", d.downloadTemplate)
	// 导入
	web.RegisterAPI("/device/{productId}/import", "POST", d.importDevice)
	// 设备导入进度
	web.RegisterAPI("/device/import-result/{token}", "GET", d.getImportResult)
}

var sseCache = sync.Map{}

func setSseData(token string, val string) {
	if len(val) == 0 {
		val = `{"success":true, "result": {"finish": false, "num": 0}}`
	}
	sseCache.Store(token, val)
}

func getSseData(token string) string {
	result, ok := sseCache.Load(token)
	if !ok {
		return ""
	}
	// `"finish": true`
	if strings.Contains(result.(string), `"finish": true`) {
		sseCache.Delete(token)
	}
	return result.(string)
}

// 导出模板
func (a *deviceApi) downloadTemplate(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(deviceResource, ImportAction) {
		return
	}
	productId := ctl.Param("productId")
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
	ctl.HeaderSet("Content-Type", "application/octet-stream")
	ctl.HeaderSet("Content-Disposition", disposition)
	ctl.HeaderSet("Content-Transfer-Encoding", "binary")
	ctl.HeaderSet("Access-Control-Expose-Headers", "Content-Disposition")
	xlsx.Write(ctl.ResponseWriter)
}

// 导入
func (a *deviceApi) importDevice(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(deviceResource, ImportAction) {
		return
	}
	productId := ctl.Param("productId")
	product, err := device.GetProductMust(productId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	if product.CreateId != ctl.GetCurrentUser().Id {
		ctl.RespError(errors.New("product is not you created"))
		return
	}
	f, _, err := ctl.FormFile("file")
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
	token := fmt.Sprintf("batch-import-device-%v", time.Now().UnixMicro())
	setSseData(token, "")
	go func() {
		var productMetaconfig map[string]bool = make(map[string]bool)
		for _, v := range product.Metaconfig {
			productMetaconfig[v.Property] = true
		}
		var devices []models.DeviceModel
		for rowIdx, row := range rows {
			if rowIdx == 0 {
				continue
			}
			dev := models.DeviceModel{Device: models.Device{Id: row[0], Name: row[1]}}
			dev.ProductId = productId
			dev.CreateId = ctl.GetCurrentUser().Id
			var devMetaconfig map[string]string = map[string]string{}
			for i := 2; i < len(row); i++ {
				head := rows[0][i]
				if _, ok := productMetaconfig[head]; ok {
					devMetaconfig[head] = row[i]
				}
			}
			dev.Metaconfig = devMetaconfig
			devices = append(devices, dev)
		}
		total := 0
		resp := `{"success":true, "result": {"finish": %v, "num": %d}}`
		for _, data := range devices {
			err := device.AddDevice(&data)
			if err == nil {
				total = total + 1
			}
			if total%5 == 0 {
				setSseData(token, fmt.Sprintf(resp, false, total))
			}
		}
		setSseData(token, fmt.Sprintf(resp, true, total))
	}()
	ctl.RespOkData(token)
}

// 设备导入进度
func (a *deviceApi) getImportResult(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	token := ctl.Param("token")
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	flusher, ok := w.(http.Flusher)
	if !ok {
		log.Panic("server not support")
	}
	id := 1
	end := "event: close\ndata: close\n\n"
	for {
		result := getSseData(token)
		if len(result) == 0 {
			fmt.Fprint(w, end) // 一定要带上data，否则无效
			break
		}
		fmt.Fprintf(w, "id: %v\n", id)
		fmt.Fprintf(w, "retry: 10000\n")
		fmt.Fprintf(w, "data: %s\n\n", result)
		if strings.Contains(result, `"finish": true`) {
			fmt.Fprint(w, end) // 一定要带上data，否则无效
			break
		}
		flusher.Flush()
		time.Sleep(1 * time.Second)
		id = id + 1
	}
	logs.Infof("ImportProcess done")
}
