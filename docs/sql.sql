CREATE TABLE IF NOT EXISTS product (
  id_ VARCHAR(32) PRIMARY KEY COMMENT '主键',
  name_ VARCHAR(64) NULL COMMENT '名称',
  type_id_ VARCHAR(32) NULL COMMENT '类型id',
  create_id_ VARCHAR(32) NULL COMMENT '创建id',
  create_time_ DATETIME NULL COMMENT '创建时间'
);

CREATE TABLE IF NOT EXISTS device (
  id_ VARCHAR(32) PRIMARY KEY COMMENT '主键',
  name_ VARCHAR(64) NULL COMMENT '名称',
  product_id_ VARCHAR(32) NULL COMMENT '产品主键',
  online_status_ VARCHAR(10) NULL COMMENT '在线状态online,offline',
  create_id_ VARCHAR(32) NULL COMMENT '创建id',
  create_time_ DATETIME NULL COMMENT '创建时间'
);