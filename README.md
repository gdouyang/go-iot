# go-iot

#### 介绍
go版本的iot管理项目，用来对接不同厂商的设备以实现统一管理的目的

#### 软件架构
软件架构说明

#### 安装教程
目录结构如下
```
goiot
    - conf
    - static
    - views
    - db
    - go-iot.exe
```

#### 使用说明

1. 下载LiteIDE
2. 执行Get命令下载依赖包
3. 执行BuildAndRun

项目使用了go-sqlite3由于go-sqlite3使用了cgo在Windows环境中需要安装gcc进行编译

#### 北向接口说明

docs/北向接口.md