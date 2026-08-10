package com.hrms.app;

import com.getcapacitor.JSObject;
import com.getcapacitor.Plugin;
import com.getcapacitor.PluginCall;
import com.getcapacitor.PluginMethod;
import com.getcapacitor.annotation.CapacitorPlugin;

// Capacitor 插件：前端 JS 与原生 SharedPreferences 之间的 token 桥。
// 前端在登录成功 / 刷新令牌 / 退出登录时调用，把 WebView 侧的 token 同步到原生层，
// 供后台 NotifyWorker 独立轮询时携带 Authorization 头。
@CapacitorPlugin(name = "TokenStore")
public class TokenStorePlugin extends Plugin {
    private TokenStore store;

    @Override
    public void load() {
        store = new TokenStore(getContext());
    }

    @PluginMethod
    public void set(PluginCall call) {
        String token = call.getString("token");
        if (token == null || token.isEmpty()) {
            call.reject("token 不能为空");
            return;
        }
        store.setToken(token);
        call.resolve();
    }

    @PluginMethod
    public void get(PluginCall call) {
        JSObject ret = new JSObject();
        ret.put("token", store.getToken());
        call.resolve(ret);
    }

    @PluginMethod
    public void clear(PluginCall call) {
        store.clear();
        call.resolve();
    }
}
