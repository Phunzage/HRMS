package com.hrms.app;

import android.content.Context;

import androidx.work.Constraints;
import androidx.work.ExistingPeriodicWorkPolicy;
import androidx.work.ExistingWorkPolicy;
import androidx.work.NetworkType;
import androidx.work.OneTimeWorkRequest;
import androidx.work.PeriodicWorkRequest;
import androidx.work.WorkManager;

import java.util.concurrent.TimeUnit;

// 后台通知轮询调度：App 退到后台后 WebView 暂停，JS 轮询失效，须原生层独立轮询。
// 方案：WorkManager 周期任务（最短 15 分钟，系统可能延迟）+ App 回到前台时即时检查一次。
public class NotifyScheduler {
    private static final String PERIODIC_WORK = "hrms_notify_periodic";
    private static final String IMMEDIATE_WORK = "hrms_notify_immediate";

    // App 启动时注册周期任务（已存在则保留，避免重复排队）；需联网约束
    public static void start(Context context) {
        Constraints constraints = new Constraints.Builder()
                .setRequiredNetworkType(NetworkType.CONNECTED)
                .build();
        PeriodicWorkRequest request = new PeriodicWorkRequest.Builder(
                NotifyWorker.class, 15, TimeUnit.MINUTES)
                .setConstraints(constraints)
                .build();
        WorkManager.getInstance(context).enqueueUniquePeriodicWork(
                PERIODIC_WORK, ExistingPeriodicWorkPolicy.KEEP, request);
    }

    // App 回到前台 / 登录态恢复时立即检查一次；REPLACE：快速连续触发只保留最新一次
    public static void checkNow(Context context) {
        OneTimeWorkRequest request = new OneTimeWorkRequest.Builder(NotifyWorker.class).build();
        WorkManager.getInstance(context).enqueueUniqueWork(
                IMMEDIATE_WORK, ExistingWorkPolicy.REPLACE, request);
    }
}
