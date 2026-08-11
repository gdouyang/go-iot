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
	ota "go-iot/pkg/models/device"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"
)

var otaResource = Resource{
	Id:   "ota-mgr",
	Name: "OTA升级",
	Sort: 30, // 侧栏：OTA升级
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
	web.RegisterAPI("/device-ota/file/page", "POST", api.filePage)
	web.RegisterAPI("/device-ota/file/add", "POST", api.fileSave)
	web.RegisterAPI("/device-ota/file/update", "PUT", api.fileSave)
	web.RegisterAPI("/device-ota/file/delete", "DELETE", api.fileDelete)

	// Subscribe to device online events
	otaQueue := make(chan struct{}, 10)
	eventbus.Subscribe(eventbus.GetOnlineTopic("*", "*"), func(msg eventbus.Message) {
		m, ok := msg.(*eventbus.OnlineMessage)
		if !ok {
			return
		}
		// Check for pending OTA logs
		logs, err := ota.GetPendingOtaLog(m.DeviceId)
		if err != nil {
			logger.Errorf("Failed to get pending ota logs: %v", err)
			return
		}
		if len(logs) > 0 {
			for _, log := range logs {
				// Execute OTA
				select {
				case otaQueue <- struct{}{}:
					go func(l models.DeviceOtaLog) {
						defer func() {
							<-otaQueue
							if r := recover(); r != nil {
								logger.Errorf("ota execute panic: %v\n%s", r, debug.Stack())
							}
						}()
						api.executeOta(&l)
					}(log)
				default:
					logger.Warnf("OTA queue is full, discard task for device: %s", log.DeviceId)
				}
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
	res, err := ota.PageOtaLog(&ob)
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
		err = ota.DeleteOtaLog(id)
		if err != nil {
			ctl.RespError(err)
			return
		}
	}
	ctl.RespOk()
}

// OTA文件列表
func (a *otaApi) filePage(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(otaResource, QueryAction) {
		return
	}
	var page models.PageQuery
	err := ctl.BindJSON(&page)
	if err != nil {
		ctl.RespError(err)
		return
	}
	res, err := ota.PageOtaFile(&page)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOkData(res)
}

// 保存OTA文件
func (a *otaApi) fileSave(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(otaResource, SaveAction) {
		return
	}
	err := r.ParseMultipartForm(100 << 20)
	if err != nil {
		ctl.RespError(err)
		return
	}

	idStr := r.FormValue("id")
	var id int
	isUpdate := false
	if len(idStr) > 0 {
		id, err = strconv.Atoi(idStr)
		if err != nil {
			ctl.RespError(err)
			return
		}
		if id > 0 {
			isUpdate = true
		}
	}

	var otaFile *models.OtaFile
	if isUpdate {
		otaFile, err = ota.GetOtaFile(int64(id))
		if err != nil {
			ctl.RespError(err)
			return
		}
		if otaFile == nil {
			ctl.RespError(errors.New("file not found"))
			return
		}
	} else {
		otaFile = &models.OtaFile{
			CreateId: ctl.GetCurrentUser().Id,
		}
	}

	file, handler, err := r.FormFile("file")
	if err == nil {
		defer file.Close()

		if isUpdate && len(otaFile.Path) > 0 {
			os.Remove(otaFile.Path)
		}

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

		otaFile.Path = savePath
		otaFile.Size = handler.Size
		otaFile.Name = handler.Filename
	} else {
		if !isUpdate {
			ctl.RespError(errors.New("file is required"))
			return
		}
	}

	name := r.FormValue("name")
	if len(name) > 0 {
		otaFile.Name = name
	}

	productId := r.FormValue("productId")
	if len(productId) > 0 {
		otaFile.ProductId = productId
	}

	if isUpdate {
		err = ota.UpdateOtaFile(otaFile)
		if err != nil {
			ctl.RespError(fmt.Errorf("update file error: %v", err))
			return
		}
	} else {
		err = ota.AddOtaFile(otaFile)
		if err != nil {
			ctl.RespError(fmt.Errorf("save file error: %v", err))
			return
		}
	}

	ctl.RespOk()
}

// 删除OTA文件
func (a *otaApi) fileDelete(w http.ResponseWriter, r *http.Request) {
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
		otaFile, err := ota.GetOtaFile(id)
		if err != nil {
			ctl.RespError(err)
			return
		}
		if otaFile != nil {
			os.Remove(otaFile.Path)
			err = ota.DeleteOtaFile(id)
			if err != nil {
				ctl.RespError(err)
				return
			}
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

	var savePath string
	var fileName string
	var totalSize int64

	// Check if file is uploaded
	file, handler, err := r.FormFile("file")
	if err == nil {
		defer file.Close()
		// Save file to ./files/ota
		otaDir := "./files/ota"
		if err = os.MkdirAll(otaDir, 0755); err != nil {
			ctl.RespError(fmt.Errorf("create ota dir error: %v", err))
			return
		}
		timestamp := time.Now().Format("20060102150405")
		fileName = handler.Filename
		saveFilename := fmt.Sprintf("%s_%s", timestamp, handler.Filename)
		savePath = filepath.Join(otaDir, saveFilename)

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
		info, _ := destFile.Stat()
		totalSize = info.Size()

		// 保存到数据库
		otaFile := &models.OtaFile{
			Name:     handler.Filename,
			Path:     savePath,
			Size:     handler.Size,
			CreateId: ctl.GetCurrentUser().Id,
		}
		productId := r.FormValue("productId")
		if len(productId) > 0 {
			otaFile.ProductId = productId
		}
		ota.AddOtaFile(otaFile)

	} else {
		otaFileId := r.FormValue("otaFileId")
		if len(otaFileId) > 0 {
			id, err := strconv.ParseInt(otaFileId, 10, 64)
			if err != nil {
				ctl.RespError(errors.New("invalid otaFileId"))
				return
			}
			otaFile, err := ota.GetOtaFile(id)
			if err != nil {
				ctl.RespError(err)
				return
			}
			if otaFile == nil {
				ctl.RespError(errors.New("ota file not found"))
				return
			}
			savePath = otaFile.Path
			fileName = otaFile.Name
		}
		// Check if filePath is provided
		if len(savePath) == 0 {
			ctl.RespError(errors.New("file or filePath or otaFileId is required"))
			return
		}
		info, err := os.Stat(savePath)
		if err != nil {
			ctl.RespError(fmt.Errorf("file not found: %v", err))
			return
		}
		totalSize = info.Size()
		if len(fileName) == 0 {
			fileName = filepath.Base(savePath)
		}
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

	totalChunks := int((totalSize + int64(chunkSize) - 1) / int64(chunkSize))

	// Start background process
	var logs []*models.DeviceOtaLog
	for _, devId := range deviceIds {
		devId = strings.TrimSpace(devId)
		if len(devId) == 0 {
			continue
		}

		// Get device to find product ID
		dev, err := ota.GetDevice(devId)
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
			FileName:     fileName,
			FilePath:     savePath, // Save path
			FileSize:     totalSize,
			ChunkSize:    chunkSize,
			TotalChunks:  totalChunks,
			CurrentChunk: 0,
			Status:       status,
			Timeout:      timeout,
		}
		ota.AddOtaLog(logEntry)
		if state == core.ONLINE && !isDisconnect {
			logs = append(logs, logEntry)
		}
	}

		go func() {
			defer func() {
				if r := recover(); r != nil {
					logger.Errorf("ota batch execute panic: %v\n%s", r, debug.Stack())
				}
			}()
			sem := make(chan struct{}, 10) // Limit to 10 concurrent goroutines
			for _, log := range logs {
				sem <- struct{}{} // Acquire token
				go func(l *models.DeviceOtaLog) {
					defer func() {
						<-sem // Release token
						if r := recover(); r != nil {
							logger.Errorf("ota execute panic: %v\n%s", r, debug.Stack())
						}
					}()
					a.executeOta(l)
				}(log)
			}
		}()

	ctl.RespOk()
}

func (a *otaApi) executeOta(logEntry *models.DeviceOtaLog) {
	// If it was pending, update to in_progress
	if logEntry.Status == "pending" {
		ota.UpdateOtaLogStatus(logEntry.Id, "pending", "in_progress")
		logEntry.Status = "in_progress"
	}

	// Read file
	fileBytes, err := os.ReadFile(logEntry.FilePath)
	if err != nil {
		logEntry.Status = "fail"
		logEntry.Message = fmt.Sprintf("read file error: %v", err)
		ota.UpdateOtaLog(logEntry)
		return
	}

	totalSize := len(fileBytes)
	chunkSize := logEntry.ChunkSize
	totalChunks := logEntry.TotalChunks
	timeout := logEntry.Timeout

	// 1. Send Start Flag
	startCmd := map[string]any{
		"step":        "start",
		"fileName":    logEntry.FileName,
		"fileSize":    totalSize,
		"chunkSize":   chunkSize,
		"totalChunks": totalChunks,
	}
	invokeMsg := core.FuncInvoke{
		DeviceId:   logEntry.DeviceId,
		FunctionId: core.OTA_UPDATE,
		Data:       startCmd,
		Timeout:    timeout,
	}
	commonErr := core.DoCmdInvoke(invokeMsg)
	if commonErr != nil {
		logEntry.Status = "fail"
		logEntry.Message = fmt.Sprintf("start error: %v", commonErr.Message)
		ota.UpdateOtaLog(logEntry)
		return
	}

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
			"step":        "update",
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
		ota.UpdateOtaLog(logEntry)
	}

	if sendErr != nil {
		logEntry.Status = "fail"
		logEntry.Message = sendErr.Message
	} else {
		logEntry.Status = "success"
		logEntry.Message = "done"
	}
	ota.UpdateOtaLog(logEntry)
}
