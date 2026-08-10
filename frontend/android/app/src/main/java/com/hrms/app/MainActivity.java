package com.hrms.app;

import android.os.Bundle;

import com.getcapacitor.BridgeActivity;

public class MainActivity extends BridgeActivity {
    @Override
    public void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        // 注册 TokenStore 桥接插件：前端登录后写 SharedPreferences，供后台 NotifyWorker 读 token
        registerPlugin(TokenStorePlugin.class);
        // 启动版本更新机制：延迟 5 秒首次检查 + 随机 30~60 分钟轮询
        Updater.start(this);
        // 注册后台通知轮询：WorkManager 15 分钟周期任务（App 退后台后 WebView JS 失效，须原生层轮询）
        NotifyScheduler.start(this);
    }

    @Override
    public void onResume() {
        super.onResume();
        // 标记前台：WebView JS 轮询负责弹窗，NotifyWorker 前台只推基线不重复弹
        AppState.setForeground(true);
        // 回到前台立即检查一次新通知（WebView 恢复前的空窗期补齐，缩短感知延迟）
        NotifyScheduler.checkNow(this);
    }

    @Override
    public void onPause() {
        // 退到后台：NotifyWorker 独立轮询负责弹系统通知
        AppState.setForeground(false);
        super.onPause();
    }
}
