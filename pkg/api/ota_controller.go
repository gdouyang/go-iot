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
	"os"
	"strconv"
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
	web.RegisterAPI("/device-ota/package/page", "POST", api.package_page)
	web.RegisterAPI("/device-ota/update", "POST", api.otaUpdateDevices)
	web.RegisterAPI("/device-ota/upload", "POST", api.otaPackageUpload)
	web.RegisterAPI("/device-ota/save", "POST", api.otaPackageSave)
	web.RegisterAPI("/device-ota/{id}", "DELETE", api.Delete)
	web.RegisterAPI("/device-ota/package/{id}", "DELETE", api.package_delete)
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

// 版本包上传
func (a *otaApi) otaPackageUpload(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(otaResource, SaveAction) {
		return
	}
	// 上传文件之后保存
	f, h, err := ctl.FormFile("file")
	if err != nil {
		ctl.RespError(err)
		return
	}
	defer f.Close()
	fileName := h.Filename
	index := strings.LastIndex(fileName, ".")
	if index != -1 {
		fileName = fileName[:index] + strconv.Itoa(int(time.Now().Unix())) + fileName[index:]
	}
	os.Mkdir("./files", os.ModePerm)

	dst, err := os.OpenFile("./files/"+fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
	if err != nil {
		ctl.RespError(err)
		return
	}
	defer dst.Close()

	_, err = io.CopyBuffer(dst, f, make([]byte, 1024*32))
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOkData(fileName)
}

type packageUpload struct {
	FileId    int      `json:"file"`
	Devices   []string `json:"devices"`
	ChunkSize int      `json:"chunkSize"`
	Timeout   int      `json:"timeout"`
}

// 指定设备升级
func (a *otaApi) otaUpdateDevices(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(otaResource, SaveAction) {
		return
	}
	var fileDevices packageUpload
	err := ctl.BindJSON(&fileDevices)
	if err != nil {
		ctl.RespError(err)
		return
	}
	// 开始对所有指定设备下发升级指令
	if len(fileDevices.Devices) == 0 {
		ctl.RespError(errors.New("deviceIds is required"))
		return
	}
	deviceIds := fileDevices.Devices
	chunkSize := fileDevices.ChunkSize

	// 判断设备存在
	if fileOta, err := product.GetOta(fileDevices.FileId); err != nil {
		ctl.RespError(errors.New("文件不存在"))
		return
	} else {
		filePath := fileOta.FilePath
		fileSize := fileOta.Size
		totalChunk := (fileOta.Size / chunkSize) + 1
		fileName := fileOta.FileName
		fmt.Println("开始升级 %s 对设备 %d 分片大小 %d", filePath, fileSize, chunkSize)
		ffile := "./files/" + filePath
		if content, err := os.ReadFile(ffile); err != nil {
			ctl.RespError(err)
			return
		} else {
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
					FileName:     fileName,
					FileSize:     int64(fileSize),
					ChunkSize:    chunkSize,
					TotalChunks:  totalChunk,
					CurrentChunk: 0,
					Status:       "in_progress",
				}
				product.AddOtaLog(logEntry)

				go func() {
					// Send chunks
					var sendErr *common.Err
					for i := 0; i < totalChunk; i++ {
						start := i * chunkSize
						end := start + chunkSize
						if end > fileSize {
							end = fileSize
						}
						chunkData := content[start:end]
						encodedData := hex.EncodeToString(chunkData)

						// Command payload
						cmdData := map[string]any{
							"fileName":    fileName,
							"fileSize":    fileSize,
							"chunkSize":   chunkSize,
							"chunkIndex":  i,
							"totalChunks": totalChunk,
							"data":        encodedData, // 十六进制编码
						}

						invokeMsg := core.FuncInvoke{
							DeviceId:   devId,
							FunctionId: core.OTA_UPDATE,
							Data:       cmdData,
							Timeout:    10,
						}

						// Check if we can just invoke
						if cluster.Enabled() {
							deviceOper := core.GetDevice(devId)
							if deviceOper == nil {
								continue
							}
							if deviceOper.ClusterId != cluster.GetClusterId() {
								ctl.Request.Header.Add(cluster.X_Cluster_Timeout, fmt.Sprintf("%d", 11))
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
				}()
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

type otaPackage struct {
	File     string `json:"file"`
	FileName string `json:"file_name"`
	Size     int    `json:"size"`
	Version  string `json:"version"`
	Product  string `json:"product"`
}

func (a *otaApi) otaPackageSave(w http.ResponseWriter, r *http.Request) {
	ctl := NewAuthController(w, r)
	if ctl.isForbidden(otaResource, SaveAction) {
		return
	}

	var ota otaPackage
	err := ctl.BindJSON(&ota)
	if err != nil {
		ctl.RespError(err)
		return
	}

	// Parameters
	if len(ota.File) == 0 {
		ctl.RespError(errors.New("请指定文件"))
		return
	}

	if len(ota.FileName) == 0 {
		ctl.RespError(errors.New("请指定文件"))
		return
	}

	if len(ota.Product) == 0 {
		ctl.RespError(errors.New(("请指定产品")))
		return
	}

	if ota.Size == 0 {
		ctl.RespError(errors.New("文件大小不能为0"))
		return
	}

	if len(ota.Version) == 0 {
		ctl.RespError(errors.New(("请指定版本")))
		return
	}

	packageEntry := &models.DeviceOtaPackage{
		ProductId: ota.Product,
		FileName:  ota.FileName,
		Size:      ota.Size,
		FilePath:  ota.File,
		Version:   ota.Version,
	}
	product.AddOtaPackage(packageEntry)
	ctl.RespOk()
}

// 分页查询OTA版本
func (a *otaApi) package_page(w http.ResponseWriter, r *http.Request) {
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
	res, err := product.PageOtaPackage(&ob)
	if err != nil {
		ctl.RespError(err)
	} else {
		ctl.RespOkData(res)
	}
}

// 删除OTA包
func (d *otaApi) package_delete(w http.ResponseWriter, r *http.Request) {
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
	err = product.DeleteOtaPackage(int64(_id))
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}
