package com.hrms.app;

import android.os.Bundle;

import com.getcapacitor.BridgeActivity;

public class MainActivity extends BridgeActivity {
    @Override
    public void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        // 启动版本更新机制：延迟 5 秒首次检查 + 随机 30~60 分钟轮询
        Updater.start(this);
    }
}
