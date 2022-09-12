CREATE TABLE IF NOT EXISTS i_user (
  id_ VARCHAR(32) PRIMARY KEY COMMENT '主键',
  name_ VARCHAR(64) NULL COMMENT '名称',
  password_ VARCHAR(64) NULL COMMENT '密码',
  create_id_ VARCHAR(32) NULL COMMENT '创建id',
  create_time_ DATETIME NULL COMMENT '创建时间'
);

CREATE TABLE IF NOT EXISTS i_role (
  id_ VARCHAR(32) PRIMARY KEY COMMENT '主键',
  name_ VARCHAR(64) NULL COMMENT '名称',
  create_id_ VARCHAR(32) NULL COMMENT '创建id',
  create_time_ DATETIME NULL COMMENT '创建时间'
);

CREATE TABLE IF NOT EXISTS i_user_rel_role (
  user_id_ VARCHAR(32) PRIMARY KEY COMMENT '用户id',
  role_id_ VARCHAR(32) NULL COMMENT '角色id'
);

CREATE TABLE IF NOT EXISTS i_product (
  id_ VARCHAR(32) PRIMARY KEY COMMENT '主键',
  name_ VARCHAR(64) NULL COMMENT '名称',
  type_id_ VARCHAR(32) NULL COMMENT '类型id',
  meta_data_ TEXT NULL COMMENT '物模型',
  meta_config_ TEXT NULL COMMENT '自定义配置',
  create_id_ VARCHAR(32) NULL COMMENT '创建id',
  create_time_ DATETIME NULL COMMENT '创建时间'
);

CREATE TABLE IF NOT EXISTS i_device (
  id_ VARCHAR(32) PRIMARY KEY COMMENT '主键',
  name_ VARCHAR(64) NULL COMMENT '名称',
  product_id_ VARCHAR(32) NULL COMMENT '产品主键',
  online_status_ VARCHAR(10) NULL COMMENT '在线状态online,offline',
  meta_config_ TEXT NULL COMMENT '自定义配置',
  create_id_ VARCHAR(32) NULL COMMENT '创建id',
  create_time_ DATETIME NULL COMMENT '创建时间'
);

CREATE TABLE IF NOT EXISTS i_server_port (
  id_ bigint(20) PRIMARY KEY COMMENT '主键',
  port_ int NULL COMMENT '端口号',
  product_id_ VARCHAR(32) NULL COMMENT '产品主键'
);