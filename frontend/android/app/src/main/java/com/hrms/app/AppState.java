package com.hrms.app;

// 前台状态标记：MainActivity onResume/onPause 维护。
// 用途：App 在前台时 WebView JS 轮询负责弹通知（LocalNotifications），
// 原生 NotifyWorker 若再弹会造成重复，故前台时 Worker 只静默推进基线不弹窗。
// 进程被杀后默认 false（后台 Worker 由 WorkManager 拉起时按后台逻辑处理）。
public class AppState {
    private static volatile boolean foreground = false;

    public static void setForeground(boolean fg) {
        foreground = fg;
    }

    public static boolean isForeground() {
        return foreground;
    }
}
