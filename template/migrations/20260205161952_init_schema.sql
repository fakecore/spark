-- Create "sys_login_log" table
CREATE TABLE "sys_login_log" ("id" bigserial NOT NULL, "created_by" bigint NULL, "created_at" bigint NULL, "login_name" character varying(64) NULL, "ipaddr" character varying(64) NULL, "login_location" character varying(256) NULL, "browser" character varying(128) NULL, "os" character varying(128) NULL, "status" smallint NOT NULL DEFAULT 1, "msg" character varying(512) NULL, "module" character varying(64) NULL, PRIMARY KEY ("id"));
-- Set comment to column: "created_by" on table: "sys_login_log"
COMMENT ON COLUMN "sys_login_log"."created_by" IS '创建者';
-- Set comment to column: "created_at" on table: "sys_login_log"
COMMENT ON COLUMN "sys_login_log"."created_at" IS '创建时间';
-- Set comment to column: "login_name" on table: "sys_login_log"
COMMENT ON COLUMN "sys_login_log"."login_name" IS '登录名';
-- Set comment to column: "ipaddr" on table: "sys_login_log"
COMMENT ON COLUMN "sys_login_log"."ipaddr" IS 'IP地址';
-- Set comment to column: "login_location" on table: "sys_login_log"
COMMENT ON COLUMN "sys_login_log"."login_location" IS '登录地点';
-- Set comment to column: "browser" on table: "sys_login_log"
COMMENT ON COLUMN "sys_login_log"."browser" IS '浏览器';
-- Set comment to column: "os" on table: "sys_login_log"
COMMENT ON COLUMN "sys_login_log"."os" IS '操作系统';
-- Set comment to column: "status" on table: "sys_login_log"
COMMENT ON COLUMN "sys_login_log"."status" IS '状态 0失败 1成功';
-- Set comment to column: "msg" on table: "sys_login_log"
COMMENT ON COLUMN "sys_login_log"."msg" IS '消息';
-- Set comment to column: "module" on table: "sys_login_log"
COMMENT ON COLUMN "sys_login_log"."module" IS '模块';
-- Create "sys_dict_value" table
CREATE TABLE "sys_dict_value" ("id" bigserial NOT NULL, "created_by" bigint NULL, "updated_by" bigint NULL, "deleted_by" bigint NULL, "created_at" bigint NULL, "updated_at" bigint NULL, "deleted_at" bigint NULL, "dict_code" character varying(64) NOT NULL, "sort" integer NOT NULL, "label" character varying(128) NOT NULL, "value" character varying(256) NOT NULL, "css_class" character varying(128) NULL, "list_class" character varying(128) NULL, "is_default" smallint NOT NULL, "status" smallint NOT NULL DEFAULT 1, "remark" character varying(512) NULL, PRIMARY KEY ("id"));
-- Create index "idx_sys_dict_value_dict_code" to table: "sys_dict_value"
CREATE INDEX "idx_sys_dict_value_dict_code" ON "sys_dict_value" ("dict_code");
-- Set comment to column: "created_by" on table: "sys_dict_value"
COMMENT ON COLUMN "sys_dict_value"."created_by" IS '创建者';
-- Set comment to column: "updated_by" on table: "sys_dict_value"
COMMENT ON COLUMN "sys_dict_value"."updated_by" IS '更新者';
-- Set comment to column: "deleted_by" on table: "sys_dict_value"
COMMENT ON COLUMN "sys_dict_value"."deleted_by" IS '删除者';
-- Set comment to column: "created_at" on table: "sys_dict_value"
COMMENT ON COLUMN "sys_dict_value"."created_at" IS '创建时间';
-- Set comment to column: "updated_at" on table: "sys_dict_value"
COMMENT ON COLUMN "sys_dict_value"."updated_at" IS '更新时间';
-- Set comment to column: "deleted_at" on table: "sys_dict_value"
COMMENT ON COLUMN "sys_dict_value"."deleted_at" IS '删除时间';
-- Set comment to column: "dict_code" on table: "sys_dict_value"
COMMENT ON COLUMN "sys_dict_value"."dict_code" IS '字典类型编码';
-- Set comment to column: "sort" on table: "sys_dict_value"
COMMENT ON COLUMN "sys_dict_value"."sort" IS '显示顺序';
-- Set comment to column: "label" on table: "sys_dict_value"
COMMENT ON COLUMN "sys_dict_value"."label" IS '字典标签';
-- Set comment to column: "value" on table: "sys_dict_value"
COMMENT ON COLUMN "sys_dict_value"."value" IS '字典值';
-- Set comment to column: "css_class" on table: "sys_dict_value"
COMMENT ON COLUMN "sys_dict_value"."css_class" IS '样式类';
-- Set comment to column: "list_class" on table: "sys_dict_value"
COMMENT ON COLUMN "sys_dict_value"."list_class" IS '列表样式';
-- Set comment to column: "is_default" on table: "sys_dict_value"
COMMENT ON COLUMN "sys_dict_value"."is_default" IS '是否默认';
-- Set comment to column: "status" on table: "sys_dict_value"
COMMENT ON COLUMN "sys_dict_value"."status" IS '状态 0禁用 1正常';
-- Set comment to column: "remark" on table: "sys_dict_value"
COMMENT ON COLUMN "sys_dict_value"."remark" IS '备注';
-- Create "base_models" table
CREATE TABLE "base_models" ("id" bigserial NOT NULL, "created_by" bigint NULL, "updated_by" bigint NULL, "deleted_by" bigint NULL, "created_at" bigint NULL, "updated_at" bigint NULL, "deleted_at" bigint NULL, PRIMARY KEY ("id"));
-- Set comment to column: "created_by" on table: "base_models"
COMMENT ON COLUMN "base_models"."created_by" IS '创建者';
-- Set comment to column: "updated_by" on table: "base_models"
COMMENT ON COLUMN "base_models"."updated_by" IS '更新者';
-- Set comment to column: "deleted_by" on table: "base_models"
COMMENT ON COLUMN "base_models"."deleted_by" IS '删除者';
-- Set comment to column: "created_at" on table: "base_models"
COMMENT ON COLUMN "base_models"."created_at" IS '创建时间';
-- Set comment to column: "updated_at" on table: "base_models"
COMMENT ON COLUMN "base_models"."updated_at" IS '更新时间';
-- Set comment to column: "deleted_at" on table: "base_models"
COMMENT ON COLUMN "base_models"."deleted_at" IS '删除时间';
-- Create "casbin_rule" table
CREATE TABLE "casbin_rule" ("id" bigserial NOT NULL, "ptype" character varying(64) NULL, "v0" character varying(256) NULL, "v1" character varying(256) NULL, "v2" character varying(256) NULL, "v3" character varying(256) NULL, "v4" character varying(256) NULL, "v5" character varying(256) NULL, PRIMARY KEY ("id"));
-- Set comment to column: "ptype" on table: "casbin_rule"
COMMENT ON COLUMN "casbin_rule"."ptype" IS '策略类型';
-- Set comment to column: "v0" on table: "casbin_rule"
COMMENT ON COLUMN "casbin_rule"."v0" IS 'V0';
-- Set comment to column: "v1" on table: "casbin_rule"
COMMENT ON COLUMN "casbin_rule"."v1" IS 'V1';
-- Set comment to column: "v2" on table: "casbin_rule"
COMMENT ON COLUMN "casbin_rule"."v2" IS 'V2';
-- Set comment to column: "v3" on table: "casbin_rule"
COMMENT ON COLUMN "casbin_rule"."v3" IS 'V3';
-- Set comment to column: "v4" on table: "casbin_rule"
COMMENT ON COLUMN "casbin_rule"."v4" IS 'V4';
-- Set comment to column: "v5" on table: "casbin_rule"
COMMENT ON COLUMN "casbin_rule"."v5" IS 'V5';
-- Create "sys_announcement" table
CREATE TABLE "sys_announcement" ("id" bigserial NOT NULL, "created_by" bigint NULL, "created_at" bigint NULL, "title" character varying(256) NOT NULL, "content" text NOT NULL, "url" character varying(512) NOT NULL, "status" smallint NOT NULL, PRIMARY KEY ("id"));
-- Set comment to column: "created_by" on table: "sys_announcement"
COMMENT ON COLUMN "sys_announcement"."created_by" IS '创建者';
-- Set comment to column: "created_at" on table: "sys_announcement"
COMMENT ON COLUMN "sys_announcement"."created_at" IS '创建时间';
-- Set comment to column: "title" on table: "sys_announcement"
COMMENT ON COLUMN "sys_announcement"."title" IS '标题';
-- Set comment to column: "content" on table: "sys_announcement"
COMMENT ON COLUMN "sys_announcement"."content" IS '内容';
-- Set comment to column: "url" on table: "sys_announcement"
COMMENT ON COLUMN "sys_announcement"."url" IS '图片URL';
-- Set comment to column: "status" on table: "sys_announcement"
COMMENT ON COLUMN "sys_announcement"."status" IS '状态 0未发布 1已发布';
-- Create "sys_config" table
CREATE TABLE "sys_config" ("id" bigserial NOT NULL, "created_by" bigint NULL, "updated_by" bigint NULL, "created_at" bigint NULL, "updated_at" bigint NULL, "name" character varying(128) NOT NULL, "key" character varying(128) NULL, "value" text NULL, "kind" smallint NOT NULL, "status" smallint NOT NULL DEFAULT 1, "remark" character varying(512) NULL, PRIMARY KEY ("id"));
-- Create index "idx_sys_config_key" to table: "sys_config"
CREATE UNIQUE INDEX "idx_sys_config_key" ON "sys_config" ("key");
-- Set comment to column: "created_by" on table: "sys_config"
COMMENT ON COLUMN "sys_config"."created_by" IS '创建者';
-- Set comment to column: "updated_by" on table: "sys_config"
COMMENT ON COLUMN "sys_config"."updated_by" IS '更新者';
-- Set comment to column: "created_at" on table: "sys_config"
COMMENT ON COLUMN "sys_config"."created_at" IS '创建时间';
-- Set comment to column: "updated_at" on table: "sys_config"
COMMENT ON COLUMN "sys_config"."updated_at" IS '更新时间';
-- Set comment to column: "name" on table: "sys_config"
COMMENT ON COLUMN "sys_config"."name" IS '配置名称';
-- Set comment to column: "key" on table: "sys_config"
COMMENT ON COLUMN "sys_config"."key" IS '配置键';
-- Set comment to column: "value" on table: "sys_config"
COMMENT ON COLUMN "sys_config"."value" IS '配置值';
-- Set comment to column: "kind" on table: "sys_config"
COMMENT ON COLUMN "sys_config"."kind" IS '类型 0系统 1用户';
-- Set comment to column: "status" on table: "sys_config"
COMMENT ON COLUMN "sys_config"."status" IS '状态 0禁用 1正常';
-- Set comment to column: "remark" on table: "sys_config"
COMMENT ON COLUMN "sys_config"."remark" IS '备注';
-- Create "sys_dept" table
CREATE TABLE "sys_dept" ("id" bigserial NOT NULL, "created_by" bigint NULL, "updated_by" bigint NULL, "deleted_by" bigint NULL, "created_at" bigint NULL, "updated_at" bigint NULL, "deleted_at" bigint NULL, "parent_id" bigint NOT NULL, "ancestors" character varying(512) NOT NULL, "dept_name" character varying(64) NOT NULL, "sort" integer NOT NULL, "leader" character varying(64) NULL, "phone" character varying(20) NULL, "email" character varying(128) NULL, "status" smallint NOT NULL DEFAULT 1, PRIMARY KEY ("id"));
-- Set comment to column: "created_by" on table: "sys_dept"
COMMENT ON COLUMN "sys_dept"."created_by" IS '创建者';
-- Set comment to column: "updated_by" on table: "sys_dept"
COMMENT ON COLUMN "sys_dept"."updated_by" IS '更新者';
-- Set comment to column: "deleted_by" on table: "sys_dept"
COMMENT ON COLUMN "sys_dept"."deleted_by" IS '删除者';
-- Set comment to column: "created_at" on table: "sys_dept"
COMMENT ON COLUMN "sys_dept"."created_at" IS '创建时间';
-- Set comment to column: "updated_at" on table: "sys_dept"
COMMENT ON COLUMN "sys_dept"."updated_at" IS '更新时间';
-- Set comment to column: "deleted_at" on table: "sys_dept"
COMMENT ON COLUMN "sys_dept"."deleted_at" IS '删除时间';
-- Set comment to column: "parent_id" on table: "sys_dept"
COMMENT ON COLUMN "sys_dept"."parent_id" IS '父部门ID';
-- Set comment to column: "ancestors" on table: "sys_dept"
COMMENT ON COLUMN "sys_dept"."ancestors" IS '祖级列表';
-- Set comment to column: "dept_name" on table: "sys_dept"
COMMENT ON COLUMN "sys_dept"."dept_name" IS '部门名称';
-- Set comment to column: "sort" on table: "sys_dept"
COMMENT ON COLUMN "sys_dept"."sort" IS '显示顺序';
-- Set comment to column: "leader" on table: "sys_dept"
COMMENT ON COLUMN "sys_dept"."leader" IS '负责人';
-- Set comment to column: "phone" on table: "sys_dept"
COMMENT ON COLUMN "sys_dept"."phone" IS '联系电话';
-- Set comment to column: "email" on table: "sys_dept"
COMMENT ON COLUMN "sys_dept"."email" IS '邮箱';
-- Set comment to column: "status" on table: "sys_dept"
COMMENT ON COLUMN "sys_dept"."status" IS '状态 0停用 1正常';
-- Create "sys_message" table
CREATE TABLE "sys_message" ("id" bigserial NOT NULL, "created_by" bigint NULL, "created_at" bigint NULL, "title" character varying(256) NOT NULL, "kind" smallint NOT NULL, "content" text NOT NULL, PRIMARY KEY ("id"));
-- Set comment to column: "created_by" on table: "sys_message"
COMMENT ON COLUMN "sys_message"."created_by" IS '创建者';
-- Set comment to column: "created_at" on table: "sys_message"
COMMENT ON COLUMN "sys_message"."created_at" IS '创建时间';
-- Set comment to column: "title" on table: "sys_message"
COMMENT ON COLUMN "sys_message"."title" IS '标题';
-- Set comment to column: "kind" on table: "sys_message"
COMMENT ON COLUMN "sys_message"."kind" IS '类型 0系统 1一对多';
-- Set comment to column: "content" on table: "sys_message"
COMMENT ON COLUMN "sys_message"."content" IS '内容';
-- Create "sys_menu" table
CREATE TABLE "sys_menu" ("id" bigserial NOT NULL, "created_by" bigint NULL, "updated_by" bigint NULL, "created_at" bigint NULL, "updated_at" bigint NULL, "pid" bigint NOT NULL, "name" character varying(64) NOT NULL, "title" character varying(64) NOT NULL, "icon" character varying(300) NOT NULL, "condition" character varying(256) NOT NULL, "remark" character varying(512) NULL, "menu_type" smallint NOT NULL, "weight" integer NOT NULL, "is_show" smallint NOT NULL DEFAULT 1, "path" character varying(256) NOT NULL, "component" character varying(256) NOT NULL, "is_link" smallint NOT NULL, "module_type" character varying(64) NOT NULL, "model_id" integer NOT NULL, "is_iframe" smallint NOT NULL, "is_cached" smallint NOT NULL, "redirect" character varying(256) NOT NULL, "is_affix" smallint NOT NULL, "link_url" character varying(512) NOT NULL, PRIMARY KEY ("id"));
-- Set comment to column: "created_by" on table: "sys_menu"
COMMENT ON COLUMN "sys_menu"."created_by" IS '创建者';
-- Set comment to column: "updated_by" on table: "sys_menu"
COMMENT ON COLUMN "sys_menu"."updated_by" IS '更新者';
-- Set comment to column: "created_at" on table: "sys_menu"
COMMENT ON COLUMN "sys_menu"."created_at" IS '创建时间';
-- Set comment to column: "updated_at" on table: "sys_menu"
COMMENT ON COLUMN "sys_menu"."updated_at" IS '更新时间';
-- Set comment to column: "pid" on table: "sys_menu"
COMMENT ON COLUMN "sys_menu"."pid" IS '父ID';
-- Set comment to column: "name" on table: "sys_menu"
COMMENT ON COLUMN "sys_menu"."name" IS '规则名称';
-- Set comment to column: "title" on table: "sys_menu"
COMMENT ON COLUMN "sys_menu"."title" IS '标题';
-- Set comment to column: "icon" on table: "sys_menu"
COMMENT ON COLUMN "sys_menu"."icon" IS '图标';
-- Set comment to column: "condition" on table: "sys_menu"
COMMENT ON COLUMN "sys_menu"."condition" IS '条件';
-- Set comment to column: "remark" on table: "sys_menu"
COMMENT ON COLUMN "sys_menu"."remark" IS '备注';
-- Set comment to column: "menu_type" on table: "sys_menu"
COMMENT ON COLUMN "sys_menu"."menu_type" IS '类型 0目录 1菜单 2按钮';
-- Set comment to column: "weight" on table: "sys_menu"
COMMENT ON COLUMN "sys_menu"."weight" IS '权重';
-- Set comment to column: "is_show" on table: "sys_menu"
COMMENT ON COLUMN "sys_menu"."is_show" IS '显示状态 0隐藏 1显示';
-- Set comment to column: "path" on table: "sys_menu"
COMMENT ON COLUMN "sys_menu"."path" IS '路由地址';
-- Set comment to column: "component" on table: "sys_menu"
COMMENT ON COLUMN "sys_menu"."component" IS '组件路径';
-- Set comment to column: "is_link" on table: "sys_menu"
COMMENT ON COLUMN "sys_menu"."is_link" IS '是否外链';
-- Set comment to column: "module_type" on table: "sys_menu"
COMMENT ON COLUMN "sys_menu"."module_type" IS '所属模块';
-- Set comment to column: "model_id" on table: "sys_menu"
COMMENT ON COLUMN "sys_menu"."model_id" IS '模型ID';
-- Set comment to column: "is_iframe" on table: "sys_menu"
COMMENT ON COLUMN "sys_menu"."is_iframe" IS '是否内嵌iframe';
-- Set comment to column: "is_cached" on table: "sys_menu"
COMMENT ON COLUMN "sys_menu"."is_cached" IS '是否缓存';
-- Set comment to column: "redirect" on table: "sys_menu"
COMMENT ON COLUMN "sys_menu"."redirect" IS '路由重定向';
-- Set comment to column: "is_affix" on table: "sys_menu"
COMMENT ON COLUMN "sys_menu"."is_affix" IS '是否固定';
-- Set comment to column: "link_url" on table: "sys_menu"
COMMENT ON COLUMN "sys_menu"."link_url" IS '链接地址';
-- Create "sys_file" table
CREATE TABLE "sys_file" ("id" bigserial NOT NULL, "file_name" character varying(256) NOT NULL, "file_size" bigint NOT NULL, "mime_type" character varying(128) NULL, "biz_type" smallint NOT NULL DEFAULT 1, "usage_type" character varying(64) NULL, "md5" character varying(64) NULL, "oss_bucket" character varying(128) NOT NULL, "oss_path" character varying(1024) NOT NULL, "status" smallint NOT NULL, "create_time" bigint NOT NULL, "deleted_at" bigint NULL, PRIMARY KEY ("id"));
-- Set comment to column: "file_name" on table: "sys_file"
COMMENT ON COLUMN "sys_file"."file_name" IS '文件名';
-- Set comment to column: "file_size" on table: "sys_file"
COMMENT ON COLUMN "sys_file"."file_size" IS '文件大小';
-- Set comment to column: "mime_type" on table: "sys_file"
COMMENT ON COLUMN "sys_file"."mime_type" IS 'MIME类型';
-- Set comment to column: "biz_type" on table: "sys_file"
COMMENT ON COLUMN "sys_file"."biz_type" IS '业务类型 1公开 2私有';
-- Set comment to column: "usage_type" on table: "sys_file"
COMMENT ON COLUMN "sys_file"."usage_type" IS '使用类型';
-- Set comment to column: "md5" on table: "sys_file"
COMMENT ON COLUMN "sys_file"."md5" IS 'MD5';
-- Set comment to column: "oss_bucket" on table: "sys_file"
COMMENT ON COLUMN "sys_file"."oss_bucket" IS 'OSS Bucket';
-- Set comment to column: "oss_path" on table: "sys_file"
COMMENT ON COLUMN "sys_file"."oss_path" IS 'OSS路径';
-- Set comment to column: "status" on table: "sys_file"
COMMENT ON COLUMN "sys_file"."status" IS '状态 0禁用 1已上传 2已使用 3待处理';
-- Set comment to column: "create_time" on table: "sys_file"
COMMENT ON COLUMN "sys_file"."create_time" IS '创建时间';
-- Create "base_model_no_soft_deletes" table
CREATE TABLE "base_model_no_soft_deletes" ("id" bigserial NOT NULL, "created_by" bigint NULL, "updated_by" bigint NULL, "created_at" bigint NULL, "updated_at" bigint NULL, PRIMARY KEY ("id"));
-- Set comment to column: "created_by" on table: "base_model_no_soft_deletes"
COMMENT ON COLUMN "base_model_no_soft_deletes"."created_by" IS '创建者';
-- Set comment to column: "updated_by" on table: "base_model_no_soft_deletes"
COMMENT ON COLUMN "base_model_no_soft_deletes"."updated_by" IS '更新者';
-- Set comment to column: "created_at" on table: "base_model_no_soft_deletes"
COMMENT ON COLUMN "base_model_no_soft_deletes"."created_at" IS '创建时间';
-- Set comment to column: "updated_at" on table: "base_model_no_soft_deletes"
COMMENT ON COLUMN "base_model_no_soft_deletes"."updated_at" IS '更新时间';
-- Create "base_model_create_onlies" table
CREATE TABLE "base_model_create_onlies" ("id" bigserial NOT NULL, "created_by" bigint NULL, "created_at" bigint NULL, PRIMARY KEY ("id"));
-- Set comment to column: "created_by" on table: "base_model_create_onlies"
COMMENT ON COLUMN "base_model_create_onlies"."created_by" IS '创建者';
-- Set comment to column: "created_at" on table: "base_model_create_onlies"
COMMENT ON COLUMN "base_model_create_onlies"."created_at" IS '创建时间';
-- Create "sys_dict_type" table
CREATE TABLE "sys_dict_type" ("id" bigserial NOT NULL, "created_by" bigint NULL, "updated_by" bigint NULL, "created_at" bigint NULL, "updated_at" bigint NULL, "name" character varying(128) NOT NULL, "type_code" character varying(64) NOT NULL, "status" smallint NOT NULL DEFAULT 1, "remark" character varying(512) NULL, PRIMARY KEY ("id"));
-- Create index "idx_sys_dict_type_type_code" to table: "sys_dict_type"
CREATE UNIQUE INDEX "idx_sys_dict_type_type_code" ON "sys_dict_type" ("type_code");
-- Set comment to column: "created_by" on table: "sys_dict_type"
COMMENT ON COLUMN "sys_dict_type"."created_by" IS '创建者';
-- Set comment to column: "updated_by" on table: "sys_dict_type"
COMMENT ON COLUMN "sys_dict_type"."updated_by" IS '更新者';
-- Set comment to column: "created_at" on table: "sys_dict_type"
COMMENT ON COLUMN "sys_dict_type"."created_at" IS '创建时间';
-- Set comment to column: "updated_at" on table: "sys_dict_type"
COMMENT ON COLUMN "sys_dict_type"."updated_at" IS '更新时间';
-- Set comment to column: "name" on table: "sys_dict_type"
COMMENT ON COLUMN "sys_dict_type"."name" IS '字典名称';
-- Set comment to column: "type_code" on table: "sys_dict_type"
COMMENT ON COLUMN "sys_dict_type"."type_code" IS '字典类型编码';
-- Set comment to column: "status" on table: "sys_dict_type"
COMMENT ON COLUMN "sys_dict_type"."status" IS '状态 0禁用 1正常';
-- Set comment to column: "remark" on table: "sys_dict_type"
COMMENT ON COLUMN "sys_dict_type"."remark" IS '备注';
-- Create "sys_message_text" table
CREATE TABLE "sys_message_text" ("id" bigserial NOT NULL, "created_by" bigint NULL, "created_at" bigint NULL, "title" character varying(256) NOT NULL, "content" text NOT NULL, PRIMARY KEY ("id"));
-- Set comment to column: "created_by" on table: "sys_message_text"
COMMENT ON COLUMN "sys_message_text"."created_by" IS '创建者';
-- Set comment to column: "created_at" on table: "sys_message_text"
COMMENT ON COLUMN "sys_message_text"."created_at" IS '创建时间';
-- Set comment to column: "title" on table: "sys_message_text"
COMMENT ON COLUMN "sys_message_text"."title" IS '标题';
-- Set comment to column: "content" on table: "sys_message_text"
COMMENT ON COLUMN "sys_message_text"."content" IS '内容';
-- Create "sys_message_user" table
CREATE TABLE "sys_message_user" ("id" bigserial NOT NULL, "created_by" bigint NULL, "updated_by" bigint NULL, "created_at" bigint NULL, "updated_at" bigint NULL, "message_id" bigint NOT NULL, "send_id" bigint NOT NULL, "rec_id" bigint NOT NULL, "status" smallint NOT NULL, PRIMARY KEY ("id"));
-- Create index "idx_sys_message_user_message_id" to table: "sys_message_user"
CREATE INDEX "idx_sys_message_user_message_id" ON "sys_message_user" ("message_id");
-- Create index "idx_sys_message_user_rec_id" to table: "sys_message_user"
CREATE INDEX "idx_sys_message_user_rec_id" ON "sys_message_user" ("rec_id");
-- Create index "idx_sys_message_user_send_id" to table: "sys_message_user"
CREATE INDEX "idx_sys_message_user_send_id" ON "sys_message_user" ("send_id");
-- Set comment to column: "created_by" on table: "sys_message_user"
COMMENT ON COLUMN "sys_message_user"."created_by" IS '创建者';
-- Set comment to column: "updated_by" on table: "sys_message_user"
COMMENT ON COLUMN "sys_message_user"."updated_by" IS '更新者';
-- Set comment to column: "created_at" on table: "sys_message_user"
COMMENT ON COLUMN "sys_message_user"."created_at" IS '创建时间';
-- Set comment to column: "updated_at" on table: "sys_message_user"
COMMENT ON COLUMN "sys_message_user"."updated_at" IS '更新时间';
-- Set comment to column: "message_id" on table: "sys_message_user"
COMMENT ON COLUMN "sys_message_user"."message_id" IS '消息ID';
-- Set comment to column: "send_id" on table: "sys_message_user"
COMMENT ON COLUMN "sys_message_user"."send_id" IS '发送者ID';
-- Set comment to column: "rec_id" on table: "sys_message_user"
COMMENT ON COLUMN "sys_message_user"."rec_id" IS '接收者ID';
-- Set comment to column: "status" on table: "sys_message_user"
COMMENT ON COLUMN "sys_message_user"."status" IS '状态 0未读 1已读';
-- Create "sys_oper_log" table
CREATE TABLE "sys_oper_log" ("id" bigserial NOT NULL, "created_by" bigint NULL, "title" character varying(128) NULL, "business_type" smallint NOT NULL, "method" character varying(256) NULL, "request_method" character varying(16) NULL, "operator_type" smallint NOT NULL, "oper_name" character varying(64) NULL, "dept_name" character varying(64) NULL, "oper_url" character varying(512) NULL, "oper_ip" character varying(64) NULL, "oper_location" character varying(256) NULL, "oper_param" text NULL, "error_msg" text NULL, "created_at" bigint NULL, PRIMARY KEY ("id"));
-- Set comment to column: "created_by" on table: "sys_oper_log"
COMMENT ON COLUMN "sys_oper_log"."created_by" IS '创建者';
-- Set comment to column: "title" on table: "sys_oper_log"
COMMENT ON COLUMN "sys_oper_log"."title" IS '标题';
-- Set comment to column: "business_type" on table: "sys_oper_log"
COMMENT ON COLUMN "sys_oper_log"."business_type" IS '业务类型 0其他 1新增 2修改 3删除';
-- Set comment to column: "method" on table: "sys_oper_log"
COMMENT ON COLUMN "sys_oper_log"."method" IS '方法';
-- Set comment to column: "request_method" on table: "sys_oper_log"
COMMENT ON COLUMN "sys_oper_log"."request_method" IS '请求方式';
-- Set comment to column: "operator_type" on table: "sys_oper_log"
COMMENT ON COLUMN "sys_oper_log"."operator_type" IS '操作类型 0其他 1后台 2手机';
-- Set comment to column: "oper_name" on table: "sys_oper_log"
COMMENT ON COLUMN "sys_oper_log"."oper_name" IS '操作人';
-- Set comment to column: "dept_name" on table: "sys_oper_log"
COMMENT ON COLUMN "sys_oper_log"."dept_name" IS '部门';
-- Set comment to column: "oper_url" on table: "sys_oper_log"
COMMENT ON COLUMN "sys_oper_log"."oper_url" IS '请求URL';
-- Set comment to column: "oper_ip" on table: "sys_oper_log"
COMMENT ON COLUMN "sys_oper_log"."oper_ip" IS '操作IP';
-- Set comment to column: "oper_location" on table: "sys_oper_log"
COMMENT ON COLUMN "sys_oper_log"."oper_location" IS '操作地点';
-- Set comment to column: "oper_param" on table: "sys_oper_log"
COMMENT ON COLUMN "sys_oper_log"."oper_param" IS '请求参数';
-- Set comment to column: "error_msg" on table: "sys_oper_log"
COMMENT ON COLUMN "sys_oper_log"."error_msg" IS '错误消息';
-- Create "sys_post" table
CREATE TABLE "sys_post" ("id" bigserial NOT NULL, "created_by" bigint NULL, "updated_by" bigint NULL, "deleted_by" bigint NULL, "created_at" bigint NULL, "updated_at" bigint NULL, "deleted_at" bigint NULL, "post_code" character varying(64) NOT NULL, "post_name" character varying(64) NOT NULL, "sort" integer NOT NULL, "status" smallint NOT NULL DEFAULT 1, "remark" character varying(512) NULL, PRIMARY KEY ("id"));
-- Set comment to column: "created_by" on table: "sys_post"
COMMENT ON COLUMN "sys_post"."created_by" IS '创建者';
-- Set comment to column: "updated_by" on table: "sys_post"
COMMENT ON COLUMN "sys_post"."updated_by" IS '更新者';
-- Set comment to column: "deleted_by" on table: "sys_post"
COMMENT ON COLUMN "sys_post"."deleted_by" IS '删除者';
-- Set comment to column: "created_at" on table: "sys_post"
COMMENT ON COLUMN "sys_post"."created_at" IS '创建时间';
-- Set comment to column: "updated_at" on table: "sys_post"
COMMENT ON COLUMN "sys_post"."updated_at" IS '更新时间';
-- Set comment to column: "deleted_at" on table: "sys_post"
COMMENT ON COLUMN "sys_post"."deleted_at" IS '删除时间';
-- Set comment to column: "post_code" on table: "sys_post"
COMMENT ON COLUMN "sys_post"."post_code" IS '岗位编码';
-- Set comment to column: "post_name" on table: "sys_post"
COMMENT ON COLUMN "sys_post"."post_name" IS '岗位名称';
-- Set comment to column: "sort" on table: "sys_post"
COMMENT ON COLUMN "sys_post"."sort" IS '显示顺序';
-- Set comment to column: "status" on table: "sys_post"
COMMENT ON COLUMN "sys_post"."status" IS '状态 0停用 1正常';
-- Set comment to column: "remark" on table: "sys_post"
COMMENT ON COLUMN "sys_post"."remark" IS '备注';
-- Create "sys_role" table
CREATE TABLE "sys_role" ("id" bigserial NOT NULL, "created_by" bigint NULL, "updated_by" bigint NULL, "created_at" bigint NULL, "updated_at" bigint NULL, "status" smallint NOT NULL, "sort" integer NOT NULL, "name" character varying(64) NOT NULL, "data_scope" smallint NOT NULL DEFAULT 3, "remark" character varying(512) NULL, PRIMARY KEY ("id"));
-- Set comment to column: "created_by" on table: "sys_role"
COMMENT ON COLUMN "sys_role"."created_by" IS '创建者';
-- Set comment to column: "updated_by" on table: "sys_role"
COMMENT ON COLUMN "sys_role"."updated_by" IS '更新者';
-- Set comment to column: "created_at" on table: "sys_role"
COMMENT ON COLUMN "sys_role"."created_at" IS '创建时间';
-- Set comment to column: "updated_at" on table: "sys_role"
COMMENT ON COLUMN "sys_role"."updated_at" IS '更新时间';
-- Set comment to column: "status" on table: "sys_role"
COMMENT ON COLUMN "sys_role"."status" IS '状态 0禁用 1正常';
-- Set comment to column: "sort" on table: "sys_role"
COMMENT ON COLUMN "sys_role"."sort" IS '显示顺序';
-- Set comment to column: "name" on table: "sys_role"
COMMENT ON COLUMN "sys_role"."name" IS '角色名称';
-- Set comment to column: "data_scope" on table: "sys_role"
COMMENT ON COLUMN "sys_role"."data_scope" IS '数据范围 1全部 2自定 3本部门 4本部门及以下';
-- Set comment to column: "remark" on table: "sys_role"
COMMENT ON COLUMN "sys_role"."remark" IS '备注';
-- Create "sys_role_dept" table
CREATE TABLE "sys_role_dept" ("id" bigserial NOT NULL, "role_id" bigint NOT NULL, "dept_id" bigint NOT NULL, PRIMARY KEY ("id"));
-- Create index "idx_sys_role_dept_dept_id" to table: "sys_role_dept"
CREATE INDEX "idx_sys_role_dept_dept_id" ON "sys_role_dept" ("dept_id");
-- Create index "idx_sys_role_dept_role_id" to table: "sys_role_dept"
CREATE INDEX "idx_sys_role_dept_role_id" ON "sys_role_dept" ("role_id");
-- Set comment to column: "role_id" on table: "sys_role_dept"
COMMENT ON COLUMN "sys_role_dept"."role_id" IS '角色ID';
-- Set comment to column: "dept_id" on table: "sys_role_dept"
COMMENT ON COLUMN "sys_role_dept"."dept_id" IS '部门ID';
-- Create "sys_user_post" table
CREATE TABLE "sys_user_post" ("id" bigserial NOT NULL, "user_id" bigint NOT NULL, "post_id" bigint NOT NULL, PRIMARY KEY ("id"));
-- Create index "idx_sys_user_post_post_id" to table: "sys_user_post"
CREATE INDEX "idx_sys_user_post_post_id" ON "sys_user_post" ("post_id");
-- Create index "idx_sys_user_post_user_id" to table: "sys_user_post"
CREATE INDEX "idx_sys_user_post_user_id" ON "sys_user_post" ("user_id");
-- Set comment to column: "user_id" on table: "sys_user_post"
COMMENT ON COLUMN "sys_user_post"."user_id" IS '用户ID';
-- Set comment to column: "post_id" on table: "sys_user_post"
COMMENT ON COLUMN "sys_user_post"."post_id" IS '岗位ID';
-- Create "sys_user_o_auth" table
CREATE TABLE "sys_user_o_auth" ("id" bigserial NOT NULL, "uid" bigint NOT NULL, "oauth_type" character varying(32) NOT NULL, "oauth_id" character varying(128) NOT NULL, "oauth_access_token" character varying(512) NOT NULL, "oauth_expire" bigint NOT NULL DEFAULT 86400, PRIMARY KEY ("id"));
-- Create index "idx_sys_user_o_auth_uid" to table: "sys_user_o_auth"
CREATE INDEX "idx_sys_user_o_auth_uid" ON "sys_user_o_auth" ("uid");
-- Set comment to column: "uid" on table: "sys_user_o_auth"
COMMENT ON COLUMN "sys_user_o_auth"."uid" IS '用户ID';
-- Set comment to column: "oauth_type" on table: "sys_user_o_auth"
COMMENT ON COLUMN "sys_user_o_auth"."oauth_type" IS 'OAuth类型';
-- Set comment to column: "oauth_id" on table: "sys_user_o_auth"
COMMENT ON COLUMN "sys_user_o_auth"."oauth_id" IS 'OAuth ID';
-- Set comment to column: "oauth_access_token" on table: "sys_user_o_auth"
COMMENT ON COLUMN "sys_user_o_auth"."oauth_access_token" IS 'Access Token';
-- Set comment to column: "oauth_expire" on table: "sys_user_o_auth"
COMMENT ON COLUMN "sys_user_o_auth"."oauth_expire" IS '过期时间';
-- Create "sys_user_online" table
CREATE TABLE "sys_user_online" ("id" bigserial NOT NULL, "created_by" bigint NULL, "updated_by" bigint NULL, "created_at" bigint NULL, "updated_at" bigint NULL, "uuid" character varying(64) NOT NULL, "token" character varying(512) NOT NULL, "user_name" character varying(64) NOT NULL, "ip" character varying(64) NOT NULL, "explorer" character varying(128) NOT NULL, "os" character varying(128) NOT NULL, PRIMARY KEY ("id"));
-- Set comment to column: "created_by" on table: "sys_user_online"
COMMENT ON COLUMN "sys_user_online"."created_by" IS '创建者';
-- Set comment to column: "updated_by" on table: "sys_user_online"
COMMENT ON COLUMN "sys_user_online"."updated_by" IS '更新者';
-- Set comment to column: "created_at" on table: "sys_user_online"
COMMENT ON COLUMN "sys_user_online"."created_at" IS '创建时间';
-- Set comment to column: "updated_at" on table: "sys_user_online"
COMMENT ON COLUMN "sys_user_online"."updated_at" IS '更新时间';
-- Set comment to column: "uuid" on table: "sys_user_online"
COMMENT ON COLUMN "sys_user_online"."uuid" IS 'UUID';
-- Set comment to column: "token" on table: "sys_user_online"
COMMENT ON COLUMN "sys_user_online"."token" IS 'Token';
-- Set comment to column: "user_name" on table: "sys_user_online"
COMMENT ON COLUMN "sys_user_online"."user_name" IS '用户名';
-- Set comment to column: "ip" on table: "sys_user_online"
COMMENT ON COLUMN "sys_user_online"."ip" IS 'IP';
-- Set comment to column: "explorer" on table: "sys_user_online"
COMMENT ON COLUMN "sys_user_online"."explorer" IS '浏览器';
-- Set comment to column: "os" on table: "sys_user_online"
COMMENT ON COLUMN "sys_user_online"."os" IS '操作系统';
-- Create "sys_user" table
CREATE TABLE "sys_user" ("id" bigserial NOT NULL, "created_by" bigint NULL, "updated_by" bigint NULL, "deleted_by" bigint NULL, "created_at" bigint NULL, "updated_at" bigint NULL, "deleted_at" bigint NULL, "name" character varying(64) NOT NULL, "nickname" character varying(64) NOT NULL, "mobile" character varying(20) NOT NULL, "birthday" integer NOT NULL, "password" character varying(255) NOT NULL, "status" smallint NOT NULL DEFAULT 1, "email" character varying(128) NOT NULL, "sex" smallint NOT NULL DEFAULT 0, "avatar" character varying(512) NOT NULL, "dept_id" bigint NOT NULL, "is_admin" smallint NOT NULL DEFAULT 0, "address" character varying(256) NULL, "remark" character varying(512) NULL, "last_login_ip" character varying(64) NULL, "last_login_time" bigint NULL, PRIMARY KEY ("id"), CONSTRAINT "fk_sys_dept_users" FOREIGN KEY ("dept_id") REFERENCES "sys_dept" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION);
-- Create index "idx_sys_user_dept_id" to table: "sys_user"
CREATE INDEX "idx_sys_user_dept_id" ON "sys_user" ("dept_id");
-- Set comment to column: "created_by" on table: "sys_user"
COMMENT ON COLUMN "sys_user"."created_by" IS '创建者';
-- Set comment to column: "updated_by" on table: "sys_user"
COMMENT ON COLUMN "sys_user"."updated_by" IS '更新者';
-- Set comment to column: "deleted_by" on table: "sys_user"
COMMENT ON COLUMN "sys_user"."deleted_by" IS '删除者';
-- Set comment to column: "created_at" on table: "sys_user"
COMMENT ON COLUMN "sys_user"."created_at" IS '创建时间';
-- Set comment to column: "updated_at" on table: "sys_user"
COMMENT ON COLUMN "sys_user"."updated_at" IS '更新时间';
-- Set comment to column: "deleted_at" on table: "sys_user"
COMMENT ON COLUMN "sys_user"."deleted_at" IS '删除时间';
-- Set comment to column: "name" on table: "sys_user"
COMMENT ON COLUMN "sys_user"."name" IS '用户名';
-- Set comment to column: "nickname" on table: "sys_user"
COMMENT ON COLUMN "sys_user"."nickname" IS '用户昵称';
-- Set comment to column: "mobile" on table: "sys_user"
COMMENT ON COLUMN "sys_user"."mobile" IS '手机号';
-- Set comment to column: "birthday" on table: "sys_user"
COMMENT ON COLUMN "sys_user"."birthday" IS '生日';
-- Set comment to column: "password" on table: "sys_user"
COMMENT ON COLUMN "sys_user"."password" IS '登录密码';
-- Set comment to column: "status" on table: "sys_user"
COMMENT ON COLUMN "sys_user"."status" IS '状态 0禁用 1正常 2未验证';
-- Set comment to column: "email" on table: "sys_user"
COMMENT ON COLUMN "sys_user"."email" IS '邮箱';
-- Set comment to column: "sex" on table: "sys_user"
COMMENT ON COLUMN "sys_user"."sex" IS '性别 0保密 1男 2女';
-- Set comment to column: "avatar" on table: "sys_user"
COMMENT ON COLUMN "sys_user"."avatar" IS '头像';
-- Set comment to column: "dept_id" on table: "sys_user"
COMMENT ON COLUMN "sys_user"."dept_id" IS '部门ID';
-- Set comment to column: "is_admin" on table: "sys_user"
COMMENT ON COLUMN "sys_user"."is_admin" IS '是否管理员';
-- Set comment to column: "address" on table: "sys_user"
COMMENT ON COLUMN "sys_user"."address" IS '联系地址';
-- Set comment to column: "remark" on table: "sys_user"
COMMENT ON COLUMN "sys_user"."remark" IS '备注';
-- Set comment to column: "last_login_ip" on table: "sys_user"
COMMENT ON COLUMN "sys_user"."last_login_ip" IS '最后登录IP';
-- Set comment to column: "last_login_time" on table: "sys_user"
COMMENT ON COLUMN "sys_user"."last_login_time" IS '最后登录时间';
