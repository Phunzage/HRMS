package com.hrms.app;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.Context;
import android.content.Intent;
import android.content.SharedPreferences;
import android.os.Build;
import android.util.Log;

import androidx.annotation.NonNull;
import androidx.work.Worker;
import androidx.work.WorkerParameters;

import org.json.JSONArray;
import org.json.JSONObject;

import java.io.BufferedReader;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URL;

// 后台通知轮询 Worker：App 退到后台/锁屏后 WebView 暂停执行 JS（前台 JS 轮询失效），
// 由 WorkManager 周期拉起本 Worker 独立轮询，实现「不打开 App 也能收到系统通知」。
// 流程：读 SharedPreferences token → GET /notifications/recent?limit=1（带 Authorization）→
//       对比上次通知 id → 有新通知且处于后台才弹系统通知。
// 防重复/防误报设计：
//   - App 在前台时（WebView JS 轮询已负责弹 LocalNotifications）只静默推进基线，不弹，避免重复；
//   - token 变化（首次登录/切换账号/令牌刷新）时先记基线不弹，避免把历史通知当新通知误报。
public class NotifyWorker extends Worker {
    private static final String TAG = "HRMSNotify";
    // 生产服务器地址（与前端 VITE_API_BASE 一致）
    private static final String API_BASE = "https://hrms.phunzage.fun";
    // 状态存储：token 基线（识别登录会话）+ 最近通知 id 基线
    private static final String PREF = "hrms_notify_state";
    private static final String KEY_LAST_TOKEN = "last_token";
    private static final String KEY_LAST_ID = "last_notify_id";
    // 系统通知渠道（Android 8+ 必须）
    private static final String CHANNEL_ID = "hrms_notify";
    private static final String CHANNEL_NAME = "HRMS 通知";

    public NotifyWorker(@NonNull Context context, @NonNull WorkerParameters params) {
        super(context, params);
    }

    @NonNull
    @Override
    public Result doWork() {
        Context ctx = getApplicationContext();
        String token = new TokenStore(ctx).getToken();
        if (token.isEmpty()) {
            return Result.success(); // 未登录，无需轮询
        }
        try {
            JSONObject latest = fetchRecent(token);
            if (latest == null) return Result.success(); // 请求失败/无未读，下个周期再试

            long latestId = latest.optLong("id");
            if (latestId <= 0) return Result.success();

            SharedPreferences sp = ctx.getSharedPreferences(PREF, Context.MODE_PRIVATE);
            String lastToken = sp.getString(KEY_LAST_TOKEN, null);
            long lastId = sp.getLong(KEY_LAST_ID, 0);

            // 首次运行 / token 变化（登录、切换账号、刷新）：只记基线不弹，避免历史通知误报
            if (lastToken == null || !lastToken.equals(token)) {
                sp.edit().putString(KEY_LAST_TOKEN, token).putLong(KEY_LAST_ID, latestId).apply();
                return Result.success();
            }
            // App 在前台：WebView JS 轮询负责弹窗，这里只推进基线，防止重复通知
            if (AppState.isForeground()) {
                if (latestId > lastId) {
                    sp.edit().putLong(KEY_LAST_ID, latestId).apply();
                }
                return Result.success();
            }
            // 后台且出现新通知：弹系统通知
            if (latestId > lastId) {
                sp.edit().putLong(KEY_LAST_ID, latestId).apply();
                showNotification(ctx, (int) latestId, latest.optString("title"));
                Log.i(TAG, "后台新通知 id=" + latestId + " title=" + latest.optString("title"));
            }
            return Result.success();
        } catch (Exception e) {
            Log.w(TAG, "通知轮询失败：" + e.getMessage());
            // 失败静默：等下一个周期（15 分钟）或 onResume 即时检查再试
            return Result.success();
        }
    }

    // 请求最近 1 条未读通知，返回 data[0]；请求失败/无未读返回 null
    private JSONObject fetchRecent(String token) {
        try {
            URL url = new URL(API_BASE + "/api/v1/notifications/recent?limit=1");
            HttpURLConnection conn = (HttpURLConnection) url.openConnection();
            conn.setConnectTimeout(15000);
            conn.setReadTimeout(15000);
            conn.setRequestMethod("GET");
            conn.setRequestProperty("Authorization", "Bearer " + token);
            int code = conn.getResponseCode();
            if (code != 200) {
                conn.disconnect();
                return null;
            }
            JSONObject body = new JSONObject(readStream(conn.getInputStream()));
            conn.disconnect();
            if (body.optInt("code") != 0) return null;
            JSONArray arr = body.optJSONArray("data");
            if (arr == null || arr.length() == 0) return null;
            return arr.optJSONObject(0);
        } catch (Exception e) {
            Log.w(TAG, "拉取最近通知失败：" + e.getMessage());
            return null;
        }
    }

    private static String readStream(InputStream is) throws Exception {
        BufferedReader reader = new BufferedReader(new InputStreamReader(is, "UTF-8"));
        StringBuilder sb = new StringBuilder();
        String line;
        while ((line = reader.readLine()) != null) {
            sb.append(line);
        }
        reader.close();
        return sb.toString();
    }

    // 发系统通知：点击打开 MainActivity；渠道不存在时先创建
    private static void showNotification(Context ctx, int id, String text) {
        NotificationManager nm = (NotificationManager) ctx.getSystemService(Context.NOTIFICATION_SERVICE);
        if (nm == null) return;
        if (Build.VERSION.SDK_INT >= 26) {
            NotificationChannel channel = new NotificationChannel(
                    CHANNEL_ID, CHANNEL_NAME, NotificationManager.IMPORTANCE_HIGH);
            channel.setDescription("HRMS 请假审批等新通知");
            nm.createNotificationChannel(channel);
        }
        Intent intent = new Intent(ctx, MainActivity.class);
        intent.setFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP | Intent.FLAG_ACTIVITY_CLEAR_TOP);
        PendingIntent pi = PendingIntent.getActivity(ctx, 200, intent,
                PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE);
        Notification.Builder builder = Build.VERSION.SDK_INT >= 26
                ? new Notification.Builder(ctx, CHANNEL_ID)
                : new Notification.Builder(ctx);
        Notification n = builder
                .setSmallIcon(android.R.drawable.stat_notify_more)
                .setContentTitle("HRMS 新通知")
                .setContentText(text)
                .setContentIntent(pi)
                .setAutoCancel(true)
                .setPriority(Notification.PRIORITY_HIGH)
                .build();
        nm.notify(id, n);
    }
}
