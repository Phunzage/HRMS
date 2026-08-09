package com.hrms.app;

import android.app.DownloadManager;
import android.app.Notification;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.content.SharedPreferences;
import android.database.Cursor;
import android.net.Uri;
import android.os.Build;
import android.os.Environment;
import android.provider.Settings;
import android.util.Log;

// 更新下载/安装接收器：
//   1. ACTION_TRIGGER_DOWNLOAD —— 用户点击「发现新版本」通知，用 DownloadManager 下载 APK；
//   2. DOWNLOAD_COMPLETE —— 系统下载完成广播，弹出「点击安装」通知；
//   3. ACTION_TRIGGER_INSTALL —— 用户点击「点击安装」，检查未知来源权限后 ACTION_VIEW 触发安装。
public class UpdateReceiver extends BroadcastReceiver {
    private static final String TAG = "HRMSUpdater";
    private static final String PREF = "update_prefs";
    private static final String KEY_DOWNLOAD_ID = "download_id";
    private static final String KEY_URL = "latest_url";
    private static final int NOTIFY_DOWNLOADED = 1002;

    @Override
    public void onReceive(Context context, Intent intent) {
        String action = intent.getAction();
        if (Updater.ACTION_TRIGGER_DOWNLOAD.equals(action)) {
            startDownload(context);
        } else if (DownloadManager.ACTION_DOWNLOAD_COMPLETE.equals(action)) {
            onDownloadComplete(context, intent);
        } else if (Updater.ACTION_TRIGGER_INSTALL.equals(action)) {
            triggerInstall(context);
        }
    }

    // 用 DownloadManager 下载最新 APK
    private void startDownload(Context context) {
        // 优先用内存中的最新版本缓存；进程重启后从 SharedPreferences 恢复
        String url = Updater.latest != null ? Updater.latest.url : "";
        if (url.isEmpty()) {
            url = context.getSharedPreferences(PREF, Context.MODE_PRIVATE)
                    .getString(KEY_URL, "");
        }
        if (url.isEmpty()) {
            Log.w(TAG, "下载地址为空，无法下载");
            return;
        }
        context.getSharedPreferences(PREF, Context.MODE_PRIVATE)
                .edit().putString(KEY_URL, url).apply();

        try {
            DownloadManager dm = (DownloadManager) context.getSystemService(Context.DOWNLOAD_SERVICE);
            if (dm == null) return;
            DownloadManager.Request req = new DownloadManager.Request(Uri.parse(url));
            req.setTitle("HRMS 更新包下载");
            req.setDescription("正在下载 HRMS 新版本…");
            // 下载进度显示在系统通知栏，完成后仍发一条通知便于点击安装
            req.setNotificationVisibility(DownloadManager.Request.VISIBILITY_VISIBLE_NOTIFY_COMPLETED);
            req.setMimeType("application/vnd.android.package-archive");
            req.setAllowedOverMetered(true);
            req.setDestinationInExternalFilesDir(context, Environment.DIRECTORY_DOWNLOADS, "hrms-android.apk");
            long id = dm.enqueue(req);
            context.getSharedPreferences(PREF, Context.MODE_PRIVATE)
                    .edit().putLong(KEY_DOWNLOAD_ID, id).apply();
            Log.i(TAG, "已开始下载更新：" + url);
        } catch (Exception e) {
            Log.e(TAG, "启动下载失败：" + e.getMessage(), e);
        }
    }

    // 下载完成：仅响应本应用发起的下载（按 download id 匹配）
    private void onDownloadComplete(Context context, Intent intent) {
        long id = intent.getLongExtra(DownloadManager.EXTRA_DOWNLOAD_ID, -1);
        long savedId = context.getSharedPreferences(PREF, Context.MODE_PRIVATE)
                .getLong(KEY_DOWNLOAD_ID, -1);
        if (id != savedId) return;

        DownloadManager dm = (DownloadManager) context.getSystemService(Context.DOWNLOAD_SERVICE);
        if (dm == null) return;
        DownloadManager.Query q = new DownloadManager.Query().setFilterById(id);
        try (Cursor c = dm.query(q)) {
            if (c != null && c.moveToFirst()) {
                int status = c.getInt(c.getColumnIndexOrThrow(DownloadManager.COLUMN_STATUS));
                if (status == DownloadManager.STATUS_SUCCESSFUL) {
                    showInstallNotification(context);
                } else {
                    Log.w(TAG, "更新包下载未成功，status=" + status);
                }
            }
        } catch (Exception e) {
            Log.e(TAG, "查询下载状态失败：" + e.getMessage(), e);
        }
    }

    // 弹出「已下载，点击安装」通知
    private void showInstallNotification(Context context) {
        NotificationManager nm = (NotificationManager) context.getSystemService(Context.NOTIFICATION_SERVICE);
        if (nm == null) return;
        Intent install = new Intent(context, UpdateReceiver.class);
        install.setAction(Updater.ACTION_TRIGGER_INSTALL);
        PendingIntent pi = PendingIntent.getBroadcast(context, 101, install,
                PendingIntent.FLAG_UPDATE_CURRENT | PendingIntent.FLAG_IMMUTABLE);
        Notification.Builder builder = Build.VERSION.SDK_INT >= 26
                ? new Notification.Builder(context, Updater.CHANNEL_ID)
                : new Notification.Builder(context);
        Notification n = builder
                .setSmallIcon(android.R.drawable.stat_sys_download_done)
                .setContentTitle("HRMS 新版本已下载")
                .setContentText("点击安装更新")
                .setContentIntent(pi)
                .setAutoCancel(true)
                .setPriority(Notification.PRIORITY_HIGH)
                .build();
        nm.notify(NOTIFY_DOWNLOADED, n);
    }

    // 触发安装：Android 8.0+ 需「安装未知应用」授权，未授权则引导去系统设置页
    private void triggerInstall(Context context) {
        if (Build.VERSION.SDK_INT >= 26
                && !context.getPackageManager().canRequestPackageInstalls()) {
            Log.w(TAG, "未授予「安装未知应用」权限，引导用户授权");
            Intent settings = new Intent(Settings.ACTION_MANAGE_UNKNOWN_APP_SOURCES,
                    Uri.parse("package:" + context.getPackageName()));
            settings.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
            context.startActivity(settings);
            return;
        }
        long id = context.getSharedPreferences(PREF, Context.MODE_PRIVATE)
                .getLong(KEY_DOWNLOAD_ID, -1);
        if (id < 0) return;
        try {
            DownloadManager dm = (DownloadManager) context.getSystemService(Context.DOWNLOAD_SERVICE);
            if (dm == null) return;
            Uri uri = dm.getUriForDownloadedFile(id);
            if (uri == null) {
                Log.w(TAG, "无法获取下载文件 URI");
                return;
            }
            // ACTION_VIEW + 下载管理器 content URI，携带读权限即可触发系统安装器
            Intent install = new Intent(Intent.ACTION_VIEW);
            install.setDataAndType(uri, "application/vnd.android.package-archive");
            install.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
            install.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION);
            context.startActivity(install);
            Log.i(TAG, "已触发安装：" + uri);
        } catch (Exception e) {
            Log.e(TAG, "触发安装失败：" + e.getMessage(), e);
        }
    }
}
