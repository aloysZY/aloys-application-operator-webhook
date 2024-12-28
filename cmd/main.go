/*
Copyright 2024 Aloys.Zhou.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"crypto/tls"
	"flag"
	"os"

	appv1 "github.com/aloys.zy/aloys-application-operator-webhook/api/v1"
	ubzap "go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	"github.com/aloys.zy/aloys-application-operator-webhook/internal/controller"
	webhookappsv1 "github.com/aloys.zy/aloys-application-operator-webhook/internal/webhook/v1"
	// +kubebuilder:scaffold:imports
)

var (
	// scheme 它提供了 Kinds 与对应的 Go Type 的映射，即给定了 Go Type，就能够知道它的 GKV(Group Kind Verision)，这也是 Kubernetes 所有资源的注册模式
	scheme = runtime.NewScheme()
	// 日志初始化
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	// Scheme 绑定内置资源
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	// Scheme 绑定自建 CRD
	utilruntime.Must(appv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var secureMetrics bool
	var enableHTTP2 bool
	var tlsOpts []func(*tls.Config)
	flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")

	// 定义自定义的 Zap 选项
	opts := zap.Options{
		Development:     false,                                   // 生产环境模式
		Level:           ubzap.NewAtomicLevelAt(ubzap.InfoLevel), // 设置日志级别为 Info
		StacktraceLevel: ubzap.ErrorLevel,                        // 只在 Error 级别及以上添加堆栈跟踪
		// TimeEncoder:     zapcore.RFC3339TimeEncoder,              // 使用 RFC3339 格式的时间戳，全局的
		Encoder: zapcore.NewJSONEncoder(zapcore.EncoderConfig{
			TimeKey:       "ts",                          // 时间戳字段的键名
			LevelKey:      "level",                       // 日志级别字段的键名
			NameKey:       "logger",                      // 日志记录器名称字段的键名
			CallerKey:     "caller",                      // 调用者信息字段的键名
			MessageKey:    "msg",                         // 日志消息字段的键名
			StacktraceKey: "stacktrace",                  // 堆栈跟踪字段的键名
			LineEnding:    zapcore.DefaultLineEnding,     // 每行日志的换行符默认是/n
			EncodeLevel:   zapcore.LowercaseLevelEncoder, // 小写日志级别
			// 	zapcore.StringDurationEncoder: 以字符串形式表示（如 1.234s）
			// zapcore.SecondsDurationEncoder: 以秒为单位表示（如 1.234）
			// zapcore.MillisDurationEncoder: 以毫秒为单位表示（如 1234）
			// zapcore.NanosDurationEncoder: 以纳秒为单位表示（如 1234567890）
			EncodeTime:     zapcore.RFC3339TimeEncoder,     // 时间戳的编码方式（RFC3339 格式）
			EncodeDuration: zapcore.SecondsDurationEncoder, // 持续时间的编码方式（秒）
			EncodeCaller:   zapcore.ShortCallerEncoder,     // 简短的调用者信息

		}),
		DestWriter: os.Stdout, // 输出到标准输出
		ZapOpts: []ubzap.Option{
			ubzap.AddCaller(), // 添加调用者信息
		},
	}
	// 日志参数绑定到命令行
	//   -zap-devel
	//        Development Mode defaults(encoder=consoleEncoder,logLevel=Debug,stackTraceLevel=Warn). Production Mode defaults(encoder=jsonEncoder,logLevel=Info,stackTraceLevel=Error)
	//  -zap-encoder value
	//        Zap log encoding (one of 'json' or 'console')
	//  -zap-log-level value
	//        Zap Level to configure the verbosity of logging. Can be one of 'debug', 'info', 'error', or any integer value > 0 which corresponds to custom debug levels of increasing verbosity
	//  -zap-stacktrace-level value
	//        Zap Level at and above which stacktraces are captured (one of 'info', 'error', 'panic').
	//  -zap-time-encoding value
	//        Zap time encoding (one of 'epoch', 'millis', 'nano', 'iso8601', 'rfc3339' or 'rfc3339nano'). Defaults to 'epoch'.
	// 命令行参数优先：命令行参数具有最高优先级，会覆盖代码中显式定义的配置。
	// 环境变量次之：环境变量的优先级低于命令行参数，但高于代码中的默认配置。
	// 代码中显式定义的配置：这是最低优先级的配置，只有在没有命令行参数或环境变量的情况下才会生效。
	opts.BindFlags(flag.CommandLine)
	// 解析命令行参数
	flag.Parse()
	// 应用自定义选项并设置全局日志记录器
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from being vulnerable to the HTTP/2 Stream Cancellation and
	// Rapid Reset CVEs. For more information see:
	// - https://github.com/advisories/GHSA-qppj-fm5r-hxr3
	// - https://github.com/advisories/GHSA-4374-p667-p6c8
	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	var webhookServer webhook.Server
	// 修改配置，在这个基础上添加了环境变量的判断，这样在本地测试的时候传入变量即可
	webhookServer = webhook.NewServer(webhook.Options{
		// 获取证书位置,在本地测试使用
		CertDir:  "./internal/webhook/certs",
		CertName: "tls.crt",
		KeyName:  "tls.key",
		// 默认配置
		TLSOpts: tlsOpts,
	})

	// Metrics endpoint is enabled in 'config/default/kustomization.yaml'. The Metrics options configure the server.
	// More info:
	// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/metrics/server
	// - https://book.kubebuilder.io/reference/metrics.html
	metricsServerOptions := metricsserver.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		TLSOpts:       tlsOpts,
	}

	if secureMetrics {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		// These configurations ensure that only authorized users and service accounts
		// can access the metrics endpoint. The RBAC are configured in 'config/rbac/kustomization.yaml'. More info:
		// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/metrics/filters#WithAuthenticationAndAuthorization
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization

		// TODO(user): If CertDir, CertName, and KeyName are not specified, controller-runtime will automatically
		// generate self-signed certificates for the metrics server. While convenient for development and testing,
		// this setup is not recommended for production.
	}
	// mgr基本配置
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "db092cec.aloys.cn",
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	// 注册controller
	if err = (&controller.ApplicationReconciler{
		// 将 Manager 的 Client 传给 AppReconciler， (r *AppReconciler) Reconciler方法就可以使用client
		Client: mgr.GetClient(),
		// 将 Manager 的 Scheme 传给 AppReconciler， get/list获取集群信息默认是先查询Scheme
		Scheme: mgr.GetScheme(),
		// 初始化事件方法
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Application")
		os.Exit(1)
	}
	// 注册webhook
	// nolint:goconst
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = webhookappsv1.SetupApplicationWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Application")
			os.Exit(1)
		}
	}
	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
