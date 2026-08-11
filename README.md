# HRMS 人力资源管理系统

前后端分离的人力资源管理系统（HRMS），覆盖组织人事、考勤、请假审批流、薪酬、招聘与统计仪表盘等核心模块。前端使用 React 18 + TypeScript + Vite + TailwindCSS + daisyUI，后端使用 Go 1.25 + Gin + GORM v2 + PostgreSQL，基于 JWT + RBAC 实现统一认证与细粒度权限控制，使用 Docker 实现数据隔离与多公司复用。**AI项目，由 Hermes + Claude Code + DeepSeek v4 Flash 构建**

系统已上线并支持**三端形态**：
- **网页版**：浏览器访问线上服务器（移动端已初步适配，抽屉导航/表格横滑/响应式断点）
- **Windows 桌面端**：Electron 封装（loadURL 加载线上前端），支持系统通知与自动更新
- **Android 端**：Capacitor APK（远程加载模式，WebView 指向线上前端），前台通知 + 后台 WorkManager 轮询

核心亮点：**可配置审批流**（admin 在「审批流程」页配置请假/补卡的级别与审批人类型）、**系统级推送通知**（Windows 通知中心 / Android 系统通知）、**移动端初步适配**。



## 功能清单

- **认证与权限**：图形验证码登录 / JWT 双 Token（access+refresh 滚动续期）/ 登录失败锁定 / 修改密码 / RBAC 权限点控制（30 个权限点，admin / hr / manager / employee 四角色 + 自定义角色与权限勾选）

  <img src="assets/image-20260810150720897.png" alt="image-20260810150720897"  />

  <img src="assets/image-20260810150829316.png" alt="image-20260810150829316"  />

- **组织架构**：部门树形管理（负责人）、岗位管理（部门 / 级别）

- **员工档案**：员工增删改查、自动生成工号（`EMP0001`）、可同时创建登录账号、头像上传、Excel 花名册导出

  ![image-20260810150949372](assets/image-20260810150949372.png)

- **考勤管理**：员工自助打卡（自动判定迟到/早退/正常）、HR 手工补录、补卡申请审批流、考勤月报（出勤/迟到/早退/旷工/请假汇总）

  ![image-20260810151030731](assets/image-20260810151030731.png)

- **可配置审批流**：通用审批流引擎，请假/补卡统一走 `approval_flows/steps/instances/tasks` 四表；admin 可配置审批级别（1~5 级）与每级审批人类型（部门经理 / 指定角色 / 指定岗位 / 按权限审批），默认两级（部门经理 → HR 角色池终审）

  ![image-20260810151826368](assets/image-20260810151826368.png)

- **请假审批**：提交请假（年假/病假/事假/产假/婚假/其他）→ 按配置逐级流转 → 通过/驳回，自动校验并扣减假期余额；**审批通过自动联动考勤标记请假**

  ![image-20260810151302821](assets/image-20260810151302821.png)

  <img src="assets/image-20260810151343157.png" alt="image-20260810151343157"  />

  <img src="assets/Screenshot_20260810_155423.jpg" alt="Screenshot_20260810_155423" style="zoom: 25%;" />

- **薪酬管理**：按员工+月份生成工资单、**按月批量生成**、实发工资自动计算、`draft → confirmed → paid` 状态机、员工自助查看工资条、Excel 工资表导出

  ![image-20260810151523098](assets/image-20260810151523098.png)

- **招聘管理**：招聘职位发布与候选人管理（简历跟踪、面试状态流转）

  ![image-20260810151709077](assets/image-20260810151709077.png)

- **绩效管理**：考核周期（2026-H1/H2）、评分自动映射 S/A/B/C/D 评级、`draft → submitted → approved` 状态流

  ![image-20260810151654654](assets/image-20260810151654654.png)

- **公告管理**：公告发布/置顶/搜索，全员可见，已读联动，仪表盘展示最新公告

  ![image-20260810152219789](assets/image-20260810152219789.png)

- **站内通知 + 系统推送**：请假/补卡提交与审批结果实时站内通知（顶栏铃铛未读提醒）；Windows/Android 客户端支持**系统级通知**（增量检测，30s 轮询）

  <img src="assets/image-20260810152130166.png" alt="image-20260810152130166" style="zoom: 50%;" /><img src="assets/IMG_20260810_152607.jpg" alt="IMG_20260810_152607" style="zoom: 25%;" />

- **角标提醒**：待审批数 / 未读公告 / 未读通知导航角标（30s + 窗口聚焦轮询）

  ![image-20260810152044937](assets/image-20260810152044937.png)

- **统一待办中心**：/approvals 聚合待我审批与我处理的审批任务，审批进度时间线可追踪

  ![image-20260810152417146](assets/image-20260810152417146.png)

- **系统管理**：用户管理（角色分配/重置密码/禁用/数据范围）、角色管理（权限分组勾选）、审批流程配置（审批级别/审批人）、操作审计日志（登录/增删改全留痕）

  ![image-20260810152508787](assets/image-20260810152508787.png)

- **数据范围权限**：HR 可绑定负责部门，仅可见该部门及子部门数据（ResolveScope 机制保留；当前所有 hr 账号 dept_scope_id 为空 = 全公司口径）

- **统计仪表盘**：在职人数、本月入职/离职、出勤率、待审批请假/薪酬、招聘/薪酬趋势、部门分布、最新公告

  ![image-20260810150652379](assets/image-20260810150652379.png)

- **暗色模式**：浅色/深色一键切换并持久化

  ![image-20260810150627615](assets/image-20260810150627615.png)

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go 1.25 + Gin + GORM v2 |
| 数据库 | PostgreSQL 18 |
| 认证 | JWT（golang-jwt/jwt/v5）+ bcrypt |
| 前端 | React 18 + TypeScript + Vite |
| 样式 | TailwindCSS + daisyUI |
| 状态管理 | zustand |
| 请求 | Axios（拦截器注入 Bearer Token） |
| Windows 桌面端 | Electron + electron-builder |
| Android 端 | Capacitor + WorkManager + @capacitor/local-notifications |

## 目录结构

```
HRMS/
├── backend/                    # Go 后端
│   ├── cmd/server/main.go      # 入口
│   ├── internal/
│   │   ├── config/             # 配置加载 (.env)
│   │   ├── database/           # GORM 连接 + AutoMigrate + Seed
│   │   ├── models/             # GORM 模型（24 张表）
│   │   ├── repository/         # 数据访问层
│   │   ├── service/            # 业务逻辑层
│   │   ├── handler/            # HTTP 处理器
│   │   ├── middleware/         # JWT / RBAC 中间件
│   │   ├── router/             # 路由注册
│   │   └── response/           # 统一响应封装
│   └── .env.example
├── frontend/                   # React 前端
│   ├── src/
│   │   ├── api/                # API 调用封装
│   │   ├── components/         # 通用组件
│   │   ├── pages/              # 页面组件
│   │   ├── router/             # 路由配置
│   │   ├── store/              # zustand 全局状态
│   │   ├── hooks/              # usePerm / useSystemNotify
│   │   └── layouts/            # 布局组件（含移动端抽屉侧边栏）
│   └── android/                # Capacitor Android 工程（远程加载模式）
├── desktop/                    # Electron Windows 桌面端
│   └── electron/               # main / preload / updater / notify
├── prompts/                    # Claude Code 批次提示词（开发过程产物）
├── ARCHITECTURE.md             # 架构设计文档
├── DESIGN.md                   # 系统设计文档
├── TESTING.md                  # 测试流程指南
├── PROGRESS.md                 # 构建进度报告
└── start.bat / start.sh        # 本地一键启动
```

## 环境要求

| 依赖 | 版本 |
|------|------|
| Go | 1.25+ |
| Node.js | 20+ |
| PostgreSQL | 18 |

## 快速开始（本地开发）

> 生产环境已部署至线上服务器，业务使用请直接访问服务器（见「部署与分发」）。以下为本地开发/联调方式。

### 1. 创建数据库

```sql
CREATE DATABASE hrms;
```

### 2. 启动后端

```bash
cd backend
cp .env.example .env   # 修改 DB_PASSWORD 为你的数据库密码
go run ./cmd/server
```

后端默认监听 `http://localhost:8080`，首次启动自动建表（AutoMigrate）并写入种子数据（内置角色、权限、账号、部门、岗位、示例员工、审批流程定义）。

### 3. 启动前端

```bash
cd frontend
npm install
npm run dev
```

前端默认监听 `http://localhost:5173`，Vite 已将 `/api` 代理到后端 `http://localhost:8080`。

## 部署与分发

- **服务器**：阿里云 ECS（Ubuntu 24.04，Docker 部署 PostgreSQL + Go 后端 + Nginx 反代）
- **网页版**：浏览器直接访问线上地址即可（移动端已适配）
- **Windows 桌面端**：`hrms-desktop Setup` 安装包（Electron 封装，加载线上前端，支持系统通知）
- **Android 端**：`HRMS.apk`（Capacitor 远程加载模式，WebView 加载线上前端，支持软件前台通知）
- **更新机制**：客户端启动后周期调用增量检测，有新版则下载更新包并引导安装；更新包存放于服务器 `/opt/hrms/updates/`，由 admin 在系统管理中登记 `app_versions` 记录
- **当前版本**：Windows 1.0.7 / Android 1.0.5，后续支持其它系统

## 多公司部署（Docker）

一套代码可为**多家公司**提供服务，各公司**数据完全隔离**（独立数据库 + 独立后端容器 + 独立入口），共享一个 PostgreSQL 容器与后端镜像。

### 架构

```
             Nginx（按域名/端口分发）
  phunzage.fun → 主公司 :8080
  a.phunzage.fun → 公司A :8081
  b.phunzage.fun → 公司B :8082
        │                │
  ┌─────▼─────┐    ┌─────▼─────┐
  │ 后端A容器  │    │  后端B容器  │
  │(hrms_a库) │    │(hrms_b库)  │
  └─────┬─────┘    └─────┬─────┘
        │                │
  ┌─────▼────────────────▼─────┐
  │   PostgreSQL 容器（1个）      │
  │  hrms(主) hrms_a hrms_b ... │
  └─────────────────────────────┘
```

数据隔离四层面：**数据库独立**（`hrms_<公司标识>`）、**后端进程独立**（容器+端口）、**入口独立**（域名/端口）、**前端独立**（`web-<公司标识>` 目录，相对路径 `/api` 自动走各自后端）。

### 新增一家公司（一条命令）

```bash
sudo /opt/hrms/scripts/new-company.sh <公司标识> <域名>
# 例: sudo /opt/hrms/scripts/new-company.sh acme a.phunzage.fun
```

脚本自动完成：创建数据库 `hrms_acme` → 启动后端容器（自动分配 8081 起空闲端口）→ 复制前端目录 → 生成 Nginx server 块并重载。新公司首次登录用种子账号 admin/admin123，自行配置部门/员工/审批流，与主公司互不影响。

### 运维命令

| 操作 | 命令 |
|------|------|
| 查看全部容器 | `docker ps` |
| 查看单公司日志 | `docker logs -f hrms-backend-<标识>` |
| 删除公司 | `sudo /opt/hrms/scripts/remove-company.sh <标识>` |
| 备份单公司库 | `docker exec hrms-db pg_dump -U postgres hrms_<标识> > backup.sql` |
| 恢复单公司库 | `docker exec -i hrms-db psql -U postgres -d hrms_<标识> < backup.sql` |
| 进入数据库 | `docker exec -it hrms-db psql -U postgres -d hrms_<标识>` |
| 重启全部服务 | `docker compose -f /opt/hrms/docker/docker-compose.yml restart` |
| 查看资源占用 | `docker stats` |

### 前端构建（多公司复用）

前端使用**相对路径** `/api/v1`（`frontend/.env.production` 中 `VITE_API_BASE` 为空），同源请求由 Nginx 按域名反代到各公司后端——**同一构建产物即可服务所有公司**，无需为每家公司单独打包前端。

### 说明与限制

- 公司标识仅允许字母/数字/下划线（脚本有防注入校验）
- 各公司使用同一后端镜像（`hrms-backend:latest`），升级时重新构建镜像后逐公司重启
- 当前入口为 IP+端口（主公司 :80，新公司 :8081 起）；域名（含 HTTPS）需 ICP 备案通过后启用
- 客户端连接地址构建时定死，多公司分发不同入口需在登录页支持配置服务器地址（规划中）

## 内置账号

| 账号 | 密码 | 角色 | 说明 |
|------|------|------|------|
| admin | admin123 | admin | 系统管理员：用户/角色/审计管理 + 公告 + 审批流程配置（无业务权限） |
| hr | hr123 | hr | 人事专员：全公司人事业务（员工/考勤/薪酬/招聘/绩效/公告） |
| zhangwei | 123456 | manager | 部门经理（张伟）：本部门请假/补卡一级审批 |
| liuyang | 123456 | employee | 普通员工（刘洋）：自助服务（打卡/请假/工资条/绩效） |

## API 概览

见 `DESIGN.md`

## 验证与检查

```bash
# 后端：构建 + 静态检查 + 自动化测试（65 例，需本机 hrms_test 库）
cd backend && go build ./... && go vet ./... && gofmt -l . && go test ./... -count=1

# 前端
cd frontend && npm run build
```

## 一键启动（本地）

双击 `start.bat`（或 git-bash 执行 `./start.sh`）：自动检查环境 → 启动后端(8080) + 前端(5173) → 打开浏览器。停止：关闭两个服务窗口。

## 测试账号

见上方「内置账号」表。测试要点：admin 走系统管理（用户/角色/审计/公告/审批流程配置）；hr 验证全公司人事业务（考勤补录/薪酬/招聘/审批终审）；zhangwei 验证请假与补卡一级审批；liuyang 验证自助打卡/请假/工资条/头像/暗色模式。完整测试流程（含三端）见 `TESTING.md`，系统设计见 `DESIGN.md`。
