-- 手动更新字段长度
ALTER TABLE sys_user ALTER COLUMN password TYPE varchar(255);
ALTER TABLE sys_dept ALTER COLUMN ancestors TYPE varchar(512);
ALTER TABLE sys_menu ALTER COLUMN icon TYPE varchar(300);
ALTER TABLE sys_file ALTER COLUMN oss_path TYPE varchar(1024);
ALTER TABLE sys_oper_log ADD COLUMN IF NOT EXISTS created_by bigint;
