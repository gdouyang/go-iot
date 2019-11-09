# go-iot

#### 介绍
go版本的iot管理项目

#### 软件架构
软件架构说明

#### 安装教程

编辑后直接运行可执行文件

#### 使用说明

1. 下载LiteIDE
2. 执行Get命令下载依赖包
3. 执行BuildAndRun

#### 北向接口说明

设备开关`/north/control/{deviceId}/switch` index第几个开关，status[open, close] 开关
```
[{index:0,status:"close"}]
```

调光`/north/control/{deviceId}/light` value 亮度
```
{value:100}
```