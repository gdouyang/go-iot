package api

import (
	"encoding/hex"
	"errors"
	"fmt"
	"go-iot/pkg/api/web"
	"go-iot/pkg/common"
	"go-iot/pkg/core"
	"go-iot/pkg/eventbus"
	"go-iot/pkg/logger"
	"go-iot/pkg/models"
	product "go-iot/pkg/models/device"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
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
	web.RegisterAPI("/device-ota/add", "POST", api.otaUpdate)
	web.RegisterAPI("/device-ota/delete", "DELETE", api.delete)

	// Subscribe to device online events
	eventbus.Subscribe(eventbus.GetOnlineTopic("*", "*"), func(msg eventbus.Message) {
		m, ok := msg.(*eventbus.OnlineMessage)
		if !ok {
			return
		}
		// Check for pending OTA logs
		logs, err := product.GetPendingOtaLog(m.DeviceId)
		if err != nil {
			logger.Errorf("Failed to get pending ota logs: %v", err)
			return
		}
		if len(logs) > 0 {
			for _, log := range logs {
				// Execute OTA
				go api.executeOta(&log)
			}
		}
	})
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
		return
	}
	ctl.RespOkData(res)
}

// 删除OTA日志
func (a *otaApi) delete(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(otaResource, DeleteAction) {
		return
	}
	var ids []int64
	err := ctl.BindJSON(&ids)
	if err != nil {
		ctl.RespError(err)
		return
	}
	if len(ids) == 0 {
		ctl.RespError(errors.New("ids is empty"))
		return
	}
	for _, id := range ids {
		err = product.DeleteOtaLog(id)
		if err != nil {
			ctl.RespError(err)
			return
		}
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

	// Save file to ./files/ota
	otaDir := "./files/ota"
	if err = os.MkdirAll(otaDir, 0755); err != nil {
		ctl.RespError(fmt.Errorf("create ota dir error: %v", err))
		return
	}
	timestamp := time.Now().Format("20060102150405")
	saveFilename := fmt.Sprintf("%s_%s", timestamp, handler.Filename)
	savePath := filepath.Join(otaDir, saveFilename)

	destFile, err := os.Create(savePath)
	if err != nil {
		ctl.RespError(fmt.Errorf("create file error: %v", err))
		return
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, file)
	if err != nil {
		ctl.RespError(fmt.Errorf("save file error: %v", err))
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

	// Get file size
	fileInfo, _ := destFile.Stat()
	totalSize := fileInfo.Size()
	totalChunks := int((totalSize + int64(chunkSize) - 1) / int64(chunkSize))

	// Start background process
	go func() {
		sem := make(chan struct{}, 10) // Limit to 10 concurrent goroutines

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

			// Check online status
			state := core.GetDeviceState(devId, dev.ProductId)
			isDisconnect := core.IsDeviceDisconnect(devId)
			status := "pending"
			if state == core.ONLINE && !isDisconnect {
				status = "in_progress"
			}

			// Create Log
			logEntry := &models.DeviceOtaLog{
				DeviceId:     devId,
				ProductId:    dev.ProductId,
				FileName:     handler.Filename,
				FilePath:     savePath, // Save path
				FileSize:     totalSize,
				ChunkSize:    chunkSize,
				TotalChunks:  totalChunks,
				CurrentChunk: 0,
				Status:       status,
				Timeout:      timeout,
			}
			product.AddOtaLog(logEntry)
			if state == core.ONLINE && !isDisconnect {
				sem <- struct{}{} // Acquire token
				go func(log *models.DeviceOtaLog) {
					defer func() { <-sem }() // Release token
					a.executeOta(log)
				}(logEntry)
			}
		}
	}()

	ctl.RespOk()
}

func (a *otaApi) executeOta(logEntry *models.DeviceOtaLog) {
	// If it was pending, update to in_progress
	if logEntry.Status == "pending" {
		product.UpdateOtaLogStatus(logEntry.Id, "pending", "in_progress")
		logEntry.Status = "in_progress"
	}

	// Read file
	fileBytes, err := os.ReadFile(logEntry.FilePath)
	if err != nil {
		logEntry.Status = "fail"
		logEntry.Message = fmt.Sprintf("read file error: %v", err)
		product.UpdateOtaLog(logEntry)
		return
	}

	totalSize := len(fileBytes)
	chunkSize := logEntry.ChunkSize
	totalChunks := logEntry.TotalChunks
	timeout := logEntry.Timeout

	var sendErr *common.Err
	for i := logEntry.CurrentChunk; i < totalChunks; i++ {
		start := i * chunkSize
		end := start + chunkSize
		if end > totalSize {
			end = totalSize
		}
		chunkData := fileBytes[start:end]
		encodedData := hex.EncodeToString(chunkData)

		// Command payload
		cmdData := map[string]any{
			"fileName":    logEntry.FileName,
			"fileSize":    totalSize,
			"chunkSize":   chunkSize,
			"chunkIndex":  i + 1,
			"totalChunks": totalChunks,
			"data":        encodedData,
		}

		invokeMsg := core.FuncInvoke{
			DeviceId:   logEntry.DeviceId,
			FunctionId: core.OTA_UPDATE,
			Data:       cmdData,
			Timeout:    timeout,
		}

		// Invoke
		err := core.DoCmdInvoke(invokeMsg)
		if err != nil {
			sendErr = err
			break
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
		logEntry.Message = "done"
	}
	product.UpdateOtaLog(logEntry)
}
