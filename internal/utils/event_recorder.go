package util

import (
	// "k8s.io/api/core/v1" // 包含 v1 API 对象
	corev1 "k8s.io/api/core/v1" // 包含 v1 API 对象
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
)

var eventRecorder record.EventRecorder

// InitializeEventRecorder 初始化 EventRecorder 并将其设置为全局变量
func InitializeEventRecorder(clientSet *kubernetes.Clientset) {
	// 创建一个新的event broadcaster
	broadcaster := record.NewBroadcaster()
	broadcaster.StartRecordingToSink(&v1.EventSinkImpl{Interface: clientSet.CoreV1().Events("")})

	// 获取 EventRecorder
	eventRecorder = broadcaster.NewRecorder(scheme.Scheme, corev1.EventSource{Component: "aloys-webhook"})
}

// EventRecorder 返回全局的 EventRecorder 实例
func EventRecorder() record.EventRecorder {
	return eventRecorder
}
