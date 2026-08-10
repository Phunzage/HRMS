# HRMS 测试流程指南

> 前置条件：PostgreSQL 18 服务运行中，`hrms` 数据库已存在（脚本会自动连接，缺失时后端日志会提示）
> 一键启动：双击 `start.bat`（或 git-bash 下执行 `./start.sh`），自动拉起前后端并打开浏览器
> 生产验证：线上服务器 `http://115.29.197.130`（网页版 + Windows/Android 客户端）

## 0. 启动与访问

| 项 | 值 |
|---|---|
| 前端（本地） | http://localhost:5173 |
| 后端（本地） | http://localhost:8080 （前端已配 proxy，一般无需直连） |
| 线上（网页版） | http://115.29.197.130 |
| 停止 | 关闭 `HRMS-Backend` / `HRMS-Frontend` 两个窗口（start.sh 用 Ctrl+C） |

## 1. 内置账号

| 账号 | 密码 | 角色 | 权限范围 |
|------|------|------|----------|
| admin | admin123 | admin | 系统管理：用户/角色/审计/公告/审批流程配置 + 仪表盘监督（无业务权限） |
| hr | hr123 | hr | 全公司人事业务：员工/部门/岗位/考勤/薪酬/招聘/绩效/公告 + 审批终审 |
| zhangwei | 123456 | manager | 部门经理（张伟）：本部门请假/补卡一级审批 |
| liuyang | 123456 | employee | 普通员工（刘洋）：自助打卡/请假/工资条/绩效 |

> **登录需要图形验证码**：登录页会先显示验证码图片。开发/自动化测试可用下方「验证码绕过」的 JWT 注入方式（跳过验证码，直接拿 token）。

## 2. 推荐测试路径（按角色）

### A. 管理员（admin）— 系统管理 + 审批流程配置
1. 登录后进入**仪表盘**：确认统计卡片有数据（员工总数、部门分布、趋势图）——admin 仅「监督」视图
2. **部门/岗位**：admin **无**部门/岗位/员工/考勤/薪酬/招聘等业务权限（菜单不显示，接口 403）——验证权限边界
3. **用户管理**：创建/编辑用户、分配角色、重置密码、禁用；验证 hr 账号只能由 admin 创建
4. **角色管理**：权限分组勾选。测试点：**取消 hr 的 leave:approve → 重启后端 → hr 的审批权限不回退**（seed 只在首次配置时初始化）
5. **审批流程配置**：进「审批流程」页，把「请假审批」改为 **3 级**（如 部门经理 → 指定角色 → 按权限审批）→ 保存 → 提交一笔请假 → 验证按 3 级流转；测完改回 2 级
6. **公告管理**：发布/置顶公告 → 全员可见；**审计日志**：查看登录/增删改留痕

### B. 人事（hr）— 全公司人事业务
1. **考勤**：手工补录一条记录（验证 late 判定）、补卡申请处理
2. **薪酬**：为员工建工资单并确认/发放、批量生成
3. **招聘/绩效/公告**：各模块读写正常；数据范围为**全公司**
4. **审批终审**：收到请假/补卡的 HR 池终审待办（角色池广播），点通过/驳回
5. 验证**权限**：hr **无**用户/角色/审计权限（菜单不显示、接口 403）

### C. 经理（zhangwei）— 审批 + 系统通知
1. **请假审批**：「待我审批」Tab 应看到本部门员工（liuyang）的请假申请；点「通过」→ liuyang 的余额应扣减；再测「驳回」（填原因）
2. **补卡审批**：本部门员工的补卡申请一级审批
3. **越权验证**：经理看不到其他部门的请假单；非审批人调用 approve 返回 403
4. **系统通知接收**：提交/审批动作发生后，Windows 桌面端应在 30s 内收到**系统通知**（Windows 通知中心）

### D. 员工（liuyang）— 自助 + 移动端
1. **考勤**：打卡上班/下班（9 点后打卡应显示迟到 badge）
2. **请假**：提交年假申请（如 3 天）→ 状态 pending → 让 zhangwei 审批 → 查看余额变化（年假初始 10 天）
   - 边界：申请超过剩余额度（如剩 7 天申请 8 天）应提示「余额不足」
3. **薪酬/绩效**：只能看到自己的工资单/绩效（试试改 URL/筛选，后端会过滤）
4. **个人中心**：查看本人档案、修改密码（改完用新密码重新登录）
5. **移动端抽屉导航**（手机/窄窗口）：点击汉堡按钮 → 抽屉侧边栏滑出 → 选中菜单后自动关闭；表格横向可滑动、弹窗不超出屏幕

## 3. 权限与安全抽查

| 场景 | 预期 |
|------|------|
| 未登录访问任意页面 | 跳转 /login |
| liuyang 访问 /employees、/recruitment 页面 | 菜单不显示；直接输 URL 调接口返回 403 |
| admin 访问 /employees、/salaries 页面 | 菜单不显示；接口 403（admin 无业务权限） |
| hr 访问 /users、/roles、/audit-logs | 菜单不显示；接口 403 |
| liuyang 访问 /salaries 列表 | 只返回本人记录 |
| 重复打卡 / 重复审批 / 重复驳回 | 400 中文错误提示 |
| 删除有员工/子部门的部门 | 400 拒绝删除 |
| 删除被占用的岗位 | 400 拒绝删除 |
| 非审批人调 /approvals/tasks/:id/approve | 403「无权限审批该请假申请」 |
| 取消 hr 的 leave:approve 后重启后端 | hr 不再有请假审批权限（seed 不还原） |

## 4. 系统通知测试

前置：Windows 桌面端（或 Android 端）已登录，并允许系统通知权限。

1. liuyang（网页/桌面）提交一笔请假申请
2. zhangwei（Windows 桌面端）**30s 内**（30s 轮询）应弹出 Windows 通知中心提醒，点击可聚焦应用
3. zhangwei 审批通过后，liuyang 端 30s 内收到结果通知
4. Android 端：前台打开 App 时走 LocalNotifications 即时弹；后台时由 WorkManager 15 分钟周期轮询（最坏 15min 延迟）
5. 增量检测验证：登录后不会对历史未读重复弹窗（localStorage 记录 `hrms_last_notify_id_<username>`）

## 5. 三端测试

### 网页版
- 桌面浏览器（≥1024px）：常驻侧边栏、全功能页面
- 移动浏览器 / 窄窗口（375×667）：抽屉导航、表格横滑、Modal `max-w-[95vw]`、工具栏换行——无横向溢出
- 暗色模式切换并持久化；导航角标（待审批/未读公告/未读通知）随轮询更新

### Windows 桌面端
- 安装包 `hrms-desktop Setup` 安装 → 启动加载线上前端（loadURL http://115.29.197.130）
- 系统通知（见第 4 节）；自动更新提示条（hrmsUpdater bridge）；桌面端无需单独部署前端

### Android 端
- 安装 `HRMS.apk`（com.hrms.app，**远程加载模式**，需联网）→ 启动加载线上前端
- 前台通知弹窗（Android 13+ 需授权）；后台 WorkManager 轮询；更新检测（v1.0.4 起含后台轮询）

## 6. 接口快速验证（curl）

> 登录接口现在**必填图形验证码**（captcha_id + captcha_code），需先请求验证码。以下给出两种方式：

### 方式一：先取验证码再登录（脚本化，需识别验证码）

```bash
# 1. 取验证码（返回 captcha_id + base64 图片）
CAPTCHA=$(curl -s localhost:8080/api/v1/auth/captcha)
echo "$CAPTCHA" | python -c "import json,sys;print(json.load(sys.stdin)['data']['captcha_id'])"
# captcha_id 拿到后，将 base64 图片解码识别出 4 位数字（如 ddddocr），填入 captcha_code 再登录

# 2. 登录（验证码必填）
TOKEN=$(curl -s -X POST localhost:8080/api/v1/auth/login -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"admin123","captcha_id":"<上一步的captcha_id>","captcha_code":"<识别出的4位数字>"}' \
  | python -c "import json,sys;print(json.load(sys.stdin)['data']['token'])")

# 3. 之后业务接口正常使用 TOKEN
curl -s localhost:8080/api/v1/health
curl -s localhost:8080/api/v1/dashboard/stats -H "Authorization: Bearer $TOKEN"
curl -s "localhost:8080/api/v1/employees?page=1&page_size=5" -H "Authorization: Bearer $TOKEN"
```

### 方式二：JWT 注入（开发/测试环境绕过验证码）

开发环境直接拿 `JWT_SECRET` 自签一个 HS256 token（claims 含 user_id/username/role_code）即可跳过验证码登录：

```bash
# 用后端 .env 里的 JWT_SECRET，python 脚本自签 token 注入：
# （批次 20 附录的 JWT 注入法——详见 prompts/batch20_fix_black_screen.md 附录）
TOKEN=<python 自签的 HS256 token>
curl -s "localhost:8080/api/v1/employees?page=1&page_size=5" -H "Authorization: Bearer $TOKEN"
```

## 7. 常见问题

| 现象 | 处理 |
|------|------|
| 后端窗口报数据库连接失败 | 确认 PostgreSQL 服务已启动、backend/.env 密码正确（按实际配置） |
| 前端 5173 被占用 | 关闭旧 dev server 或改 vite.config.ts 端口 |
| 登录提示「验证码错误或已过期」 | 验证码一次即消费，重新点验证码图换一个再登录 |
| 登录提示「用户名或密码错误」 | 确认使用上表账号；改过密码则用新密码 |
| 页面 401 跳回登录 | token 过期，重新登录即可（access+refresh 会自动滚动续期） |
| 数据被测试污染 | 开发库可直接重来：删除重建 hrms 库，重启后端自动重建表+种子数据 |
