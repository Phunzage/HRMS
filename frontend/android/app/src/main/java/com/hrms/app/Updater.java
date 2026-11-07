package com.hrms.app;

import android.Manifest;
import android.app.Activity;
import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.Context;
import android.content.Intent;
import android.content.pm.PackageManager;
import android.os.Build;
import android.os.Handler;
import android.os.Looper;
import android.util.Log;

import org.json.JSONObject;

import java.io.BufferedReader;
import java.io.InputStream;
import java.io.InputStreamReader;
import java.net.HttpURLConnection;
import java.net.URL;
import java.util.Random;

// Android 端版本更新检查与下载（随机时间拉取机制，与 Windows/Electron 端一致）
// 设计要点：
//   1. 应用启动后延迟 5 秒执行首次检查；
//   2. 每次检查完成后，生成 30~60 分钟随机间隔，用 Handler.postDelayed 排下次检查，
//      使所有客户端的拉取时间错开（避免同时请求服务器形成脉冲）；
//   3. 检查到新版本时弹系统通知，点击后用 DownloadManager 下载 APK；
//   4. 下载完成广播由 UpdateReceiver 接收，弹出「点击安装」通知，ACTION_VIEW 触发安装。
public class Updater {
    private static final String TAG = "HRMSUpdater";
    // 生产服务器地址（更新 API 为公开接口，无需登录）
    private static final String API_BASE = "https://hrms.phunzage.fun";
    // 通知渠道：Android 8.0+ 必须
    static final String CHANNEL_ID = "updates";

    // 首次检查延迟：启动后 5 秒
    private static final long FIRST_CHECK_DELAY = 5 * 1000;
    // 随机间隔区间：30~60 分钟
    private static final long MIN_INTERVAL = 30 * 60 * 1000;
    private static final long RANDOM_RANGE = 30 * 60 * 1000;

    // 接收器自定义动作
    public static final String ACTION_TRIGGER_DOWNLOAD = "com.hrms.app.ACTION_TRIGGER_DOWNLOAD";
    public static final String ACTION_TRIGGER_INSTALL = "com.hrms.app.ACTION_TRIGGER_INSTALL";

    private static Context appContext;
    private static Handler handler;
    // 包内可见：UpdateReceiver 触发下载时读取最新版本信息
    static LatestVersion latest; // 服务器返回的最新版本信息

    // 最新版本信息缓存（供 UpdateReceiver 触发下载时使用）
    static class LatestVersion {
        String version;
        int buildNumber;
        String url;
        String releaseNotes;
    }

    // 启动更新机制：创建通知渠道、申请通知权限、安排首次检查
    public static void start(Context context) {
        appContext = context.getApplicationContext();
        handler = new Handler(Looper.getMainLooper());
        createNotificationChannel();
        // Android 13+（API 33）动态申请通知权限
        if (Build.VERSION.SDK_INT >= 33
                && context.checkSelfPermission(Manifest.permission.POST_NOTIFICATIONS)
                        != PackageManager.PERMISSION_GRANTED) {
            if (context instanceof Activity) {
                ((Activity) context).requestPermissions(
                        new String[]{Manifest.permission.POST_NOTIFICATIONS}, 100);
            }
        }
        Log.i(TAG, "更新检查已启动，当前版本 v" + BuildConfig.VERSION_NAME
                + " (build " + BuildConfig.VERSION_CODE + ")，"
                + (FIRST_CHECK_DELAY / 1000) + " 秒后首次检查");
        handler.postDelayed(Updater::check, FIRST_CHECK_DELAY);
    }

    // 创建通知渠道（Android 8+ 必须，否则通知不显示）
    private static void createNotificationChannel() {
        if (Build.VERSION.SDK_INT < 26) return;
        NotificationManager nm = (NotificationManager) appContext.getSystemService(Context.NOTIFICATION_SERVICE);
        if (nm == null) return;
        NotificationChannel channel = new NotificationChannel(
                CHANNEL_ID, "版本更新", NotificationManager.IMPORTANCE_HIGH);
        channel.setDescription("HRMS 新版本发布与下载安装通知");
        nm.createNotificationChannel(channel);
    }

    // 执行一次更新检查（GET /api/v1/update/check?platform=android&version=&build=）
    private static void check() {
        new Thread(() -> {
            boolean hasUpdate = false;
            try {
                String url = API_BASE + "/api/v1/update/check?platform=android"
                        + "&version=" + BuildConfig.VERSION_NAME
                        + "&build=" + BuildConfig.VERSION_CODE;
                HttpURLConnection conn = (HttpURLConnection) new URL(url).openConnection();
                conn.setConnectTimeout(15000);
                conn.setReadTimeout(15000);
                conn.setRequestMethod("GET");
                if (conn.getResponseCode() == 200) {
                    JSONObject body = new JSONObject(readStream(conn.getInputStream()));
                    // 后端统一响应体 { code, message, data }
                    JSONObject data = body.optJSONObject("data");
                    if (body.optInt("code") == 0 && data != null && data.optBoolean("has_update")) {
                        JSONObject l = data.optJSONObject("latest");
                        if (l != null) {
                            latest = new LatestVersion();
                            latest.version = l.optString("version");
                            latest.buildNumber = l.optInt("build_number");
                            latest.url = l.optString("url");
                            latest.releaseNotes = l.optString("release_notes");
                            hasUpdate = true;
                        }
                    }
                }
                conn.disconnect();
            } catch (Exception e) {
                Log.w(TAG, "检查更新失败：" + e.getMessage());
            }
            if (hasUpdate && latest != null) {
                notifyUpdate();
            }
            scheduleNext();
        }).start();
    }

    // 读取输入流为字符串
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

    // 安排下一次检查：30~60 分钟随机间隔 —— 「随机时间拉取」核心
    private static void scheduleNext() {
        long delay = MIN_INTERVAL + new Random().nextInt((int) RANDOM_RANGE);
        Log.i(TAG, "本次检查完成，下次更新检查约 " + (delay / 60000) + " 分钟后");
        handler.postDelayed(Updater::check, delay);
    }

    // 发现新版本：弹系统通知，点击后触发下载
    private static void notifyUpdate() {
        NotificationManager nm = (NotificationManager) appContext.getSystemService(Context.NOTIFICATION_SERVICE);
        if (nm == null) return;
        Intent trigger = new Intent(appContext, UpdateReceiver.class);
        trigger.setAction(ACTION_TRIGGER_DOWNLOAD);
        PendingIntent pi = PendingIntent.getBroadcast(appContext, 100, trigger,
                PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE);
        Notification.Builder builder = Build.VERSION.SDK_INT >= 26
                ? new Notification.Builder(appContext, CHANNEL_ID)
                : new Notification.Builder(appContext);
        Notification n = builder
                .setSmallIcon(android.R.drawable.stat_sys_download)
                .setContentTitle("发现新版本 v" + latest.version)
                .setContentText("点击下载更新 HRMS")
                .setContentIntent(pi)
                .setAutoCancel(true)
                .setPriority(Notification.PRIORITY_HIGH)
                .build();
        nm.notify(1001, n);
    }
}
