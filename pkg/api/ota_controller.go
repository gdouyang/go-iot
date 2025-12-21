package api

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go-iot/pkg/api/web"
	"go-iot/pkg/cluster"
	"go-iot/pkg/common"
	"go-iot/pkg/core"
	"go-iot/pkg/models"
	product "go-iot/pkg/models/device"
	"io"
	"net/http"
	"strconv"
	"strings"
)

var otaResource = Resource{
	Id:   "ota-mgr",
	Name: "OTA管理",
	Action: []ResourceAction{
		QueryAction,
		DeleteAction,
		SaveAction,
	},
}

func init() {
	RegResource(otaResource)

	api := &otaApi{}

	web.RegisterAPI("/device-ota/page", "POST", api.page)
	web.RegisterAPI("/device-ota/update", "POST", api.otaUpdate)
	web.RegisterAPI("/device-ota/{id}", "DELETE", api.Delete)
}

type otaApi struct {
}

// 分页查询OTA日志
func (a *otaApi) page(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(otaResource, QueryAction) {
		return
	}
	var ob models.PageQuery
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	res, err := product.PageOtaLog(&ob)
	if err != nil {
		ctl.RespError(err)
	} else {
		ctl.RespOkData(res)
	}
}

// 删除OTA日志
func (d *otaApi) Delete(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(otaResource, DeleteAction) {
		return
	}
	otaLogId := ctl.Param("id")
	_id, err := strconv.Atoi(otaLogId)
	if err != nil {
		ctl.RespError(err)
		return
	}
	err = product.DeleteOtaLog(int64(_id))
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

// OTA升级
func (a *otaApi) otaUpdate(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(otaResource, SaveAction) {
		return
	}
	// Parse Multipart
	err := r.ParseMultipartForm(100 << 20) // 100MB limit
	if err != nil {
		ctl.RespError(err)
		return
	}

	file, handler, err := r.FormFile("file")
	if err != nil {
		ctl.RespError(err)
		return
	}
	defer file.Close()

	// Read file
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		ctl.RespError(err)
		return
	}

	// Parameters
	deviceIdsStr := r.FormValue("deviceIds")
	if len(deviceIdsStr) == 0 {
		ctl.RespError(errors.New("deviceIds is required"))
		return
	}
	deviceIds := strings.Split(deviceIdsStr, ",")

	chunkSizeStr := r.FormValue("chunkSize")
	chunkSize := 1024 // default
	if len(chunkSizeStr) > 0 {
		fmt.Sscanf(chunkSizeStr, "%d", &chunkSize)
	}

	timeoutStr := r.FormValue("timeout")
	timeout := 10 // default seconds
	if len(timeoutStr) > 0 {
		fmt.Sscanf(timeoutStr, "%d", &timeout)
	}

	// Validation
	if chunkSize <= 0 {
		chunkSize = 1024
	}

	// Total chunks
	totalSize := len(fileBytes)
	totalChunks := (totalSize + chunkSize - 1) / chunkSize

	// Start background process
	go func() {
		for _, devId := range deviceIds {
			devId = strings.TrimSpace(devId)
			if len(devId) == 0 {
				continue
			}

			// Get device to find product ID
			dev, err := product.GetDevice(devId)
			if err != nil || dev == nil {
				continue
			}

			// Create Log
			logEntry := &models.DeviceOtaLog{
				DeviceId:     devId,
				ProductId:    dev.ProductId,
				FileName:     handler.Filename,
				FileSize:     int64(totalSize),
				ChunkSize:    chunkSize,
				TotalChunks:  totalChunks,
				CurrentChunk: 0,
				Status:       "in_progress",
			}
			product.AddOtaLog(logEntry)

			// Send chunks
			var sendErr *common.Err
			for i := 0; i < totalChunks; i++ {
				start := i * chunkSize
				end := start + chunkSize
				if end > totalSize {
					end = totalSize
				}
				chunkData := fileBytes[start:end]
				encodedData := hex.EncodeToString(chunkData)

				// Command payload
				cmdData := map[string]any{
					"fileName":    handler.Filename,
					"fileSize":    totalSize,
					"chunkSize":   chunkSize,
					"chunkIndex":  i,
					"totalChunks": totalChunks,
					"data":        encodedData, // 十六进制编码
				}

				invokeMsg := core.FuncInvoke{
					DeviceId:   devId,
					FunctionId: core.OTA_UPDATE,
					Data:       cmdData,
					Timeout:    timeout,
				}

				// Check if we can just invoke
				if cluster.Enabled() {
					deviceOper := core.GetDevice(devId)
					if deviceOper == nil {
						continue
					}
					if deviceOper.ClusterId != cluster.GetClusterId() {
						ctl.Request.Header.Add(cluster.X_Cluster_Timeout, fmt.Sprintf("%d", timeout+1))
						data, _ := json.Marshal(invokeMsg)
						ctl.Request.Body = io.NopCloser(strings.NewReader(string(data)))
						resp, err := cluster.SingleInvoke(deviceOper.ClusterId, ctl.Request)
						if err != nil {
							sendErr = common.NewErr500(err.Error())
							continue
						}
						if !resp.Success {
							sendErr = common.NewErr500(resp.Msg)
							continue
						}
					} else {
						err := core.DoCmdInvoke(invokeMsg)
						if err != nil {
							sendErr = err
							continue
						}
					}
				} else {
					err := core.DoCmdInvoke(invokeMsg)
					if err != nil {
						sendErr = err
						continue
					}
				}

				// Update progress
				logEntry.CurrentChunk = i + 1
				product.UpdateOtaLog(logEntry)
			}

			if sendErr != nil {
				logEntry.Status = "fail"
				logEntry.Message = sendErr.Message
			} else {
				logEntry.Status = "success"
			}
			product.UpdateOtaLog(logEntry)
		}
	}()

	ctl.RespOk()
}
