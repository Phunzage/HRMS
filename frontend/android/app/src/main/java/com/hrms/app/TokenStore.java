package com.hrms.app;

import android.content.Context;
import android.content.SharedPreferences;

// 登录 token 存储：前端 JS 经 TokenStorePlugin 桥接写入（WebView localStorage 原生层不可见），
// 后台 NotifyWorker 从这里读 token 轮询新通知。仅存内存级明文 token（与 WebView localStorage 等价）。
public class TokenStore {
    private static final String PREF = "hrms_token_store";
    private static final String KEY_TOKEN = "token";
    private final SharedPreferences sp;

    public TokenStore(Context context) {
        sp = context.getApplicationContext().getSharedPreferences(PREF, Context.MODE_PRIVATE);
    }

    public void setToken(String token) {
        sp.edit().putString(KEY_TOKEN, token).apply();
    }

    public String getToken() {
        return sp.getString(KEY_TOKEN, "");
    }

    public void clear() {
        sp.edit().remove(KEY_TOKEN).apply();
    }
}
