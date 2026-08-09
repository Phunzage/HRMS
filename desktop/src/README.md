# desktop/src

渲染进程占位目录。

本客户端不在此目录编写渲染代码：渲染进程直接引用 `frontend` 的构建产物，
打包前由脚本把 `frontend/dist` 拷入 `../build-web/`（electron-builder 打包进 asar），
开发模式则直接加载 `http://localhost:5173`（vite dev server）。
