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

package controller

import (
	"context"
	"reflect"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	appv1 "github.com/aloys.zy/aloys-application-operator-webhook/api/v1"
)

// ApplicationReconciler reconciles a Application object
type ApplicationReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=apps.aloys.cn,resources=applications,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps.aloys.cn,resources=applications/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps.aloys.cn,resources=applications/finalizers,verbs=update

// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services/status,verbs=get

// CounterReconcileApplication 记录当前调谐的轮次
var CounterReconcileApplication int64

// GenericRequeueDuration 这个是每次重试的时间间隔（修改到配置文件中）
const GenericRequeueDuration = 1 * time.Minute

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Application object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
// 实现具体的调谐逻辑
func (r *ApplicationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	setupLog := log.FromContext(ctx).WithName("Reconcile")
	// setupLog.V(0).Info("000000") // 0是 info 不写就是0
	// setupLog.V(1).Info("11111")  // 1是 debug

	// TODO(user): your log here
	// 调谐逻辑是并发的，我设置的是10，当时多个goroutine同时运行的时候，日志比较乱，这里增加了一个100毫秒的等待，并且添加了一个当前调谐次数的打印
	// time.NewTicker 函数用于创建一个新的定时器（ticker），它会定期发送时间信号到一个通道（channel）。<-time.NewTicker(1000 * time.Millisecond).C 这一行代码的作用是从这个定时器的通道中接收时间信号。
	<-time.NewTicker(1000 * time.Millisecond).C
	CounterReconcileApplication += 1
	setupLog.Info("Starting a reconcile", "number", CounterReconcileApplication)

	// 获取 Application 对象的状态
	application := &appv1.Application{}
	if err := r.Get(ctx, req.NamespacedName, application); err != nil {
		// 如果Application不存在，直接结束即可，不管什么原因导致的Application不存在，控制器进行任何操作都没有意义
		// 如果Application后续被创建了，那么会重新出发这个调谐
		if errors.IsNotFound(err) {
			setupLog.Error(err, "Failed to get Application", "name", req.Name)
			return ctrl.Result{}, client.IgnoreNotFound(err)
		}
		// 如果是其他错误，那么打印并返回错误信息，并且进行重试
		setupLog.Error(err, "Failed to get Application,will request after a short time.", "name", req.NamespacedName)
		return ctrl.Result{RequeueAfter: GenericRequeueDuration}, err
	}
	// a := application.GetResourceVersion()

	// 多次使用这个变量，提前声明，但是都是在栈上进行的操作，感觉性能影响非常小 var result ctrl.Result var err error
	if result, err := r.reconcileDeployment(ctx, application); err != nil {
		setupLog.Error(err, "Failed to reconcile deployment.", "name", req.Name)
		return result, err
	}
	if result, err := r.reconcileService(ctx, application); err != nil {
		setupLog.Error(err, "Failed to reconcile service.", "name", req.Name)
		return result, err
	}
	// 如果没有发生任何 error，返回一个空的Result，表示没有需要重试的操作，控制器可以结束当前的 reconcile loop，并开始下一个 reconcile loop。
	setupLog.Info("Finished a reconcile", "number", CounterReconcileApplication)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
// 监听到什么事件的时候需要触发调谐，是根据这里的配置进行过滤
func (r *ApplicationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	setupLog := ctrl.Log.WithName("SetupWithManager")
	return ctrl.NewControllerManagedBy(mgr).
		// 监听到 Application 创建、更新、删除事件，返回true表示触发调谐
		For(&appv1.Application{}, builder.WithPredicates(predicate.Funcs{
			// create 是肯定要触发的
			CreateFunc: func(e event.CreateEvent) bool {
				setupLog.Info("The Application has been Created.", "name", e.Object.GetName())
				return true
			},
			//
			UpdateFunc: func(e event.UpdateEvent) bool {
				// 针对GetResourceVersion字段进行匹配，如果不一致进行调谐
				if e.ObjectNew.GetResourceVersion() == e.ObjectOld.GetResourceVersion() {
					return false
				}
				// 如果新旧spec字段相同也不触发调谐
				if reflect.DeepEqual(e.ObjectNew.(*appv1.Application).Spec, e.ObjectOld.(*appv1.Application).Spec) {
					return false
				}
				// 其他情况下进行调谐
				setupLog.Info("The Application has been Updated.", "name", e.ObjectNew.GetName())
				return true
				// return !reflect.DeepEqual(e.ObjectOld.GetResourceVersion(), e.ObjectNew.GetResourceVersion())
			},
			// 删除的时候不需要触发，因为有SetControllerReference子资源的存在
			DeleteFunc: func(e event.DeleteEvent) bool {
				setupLog.Info("The Application has been Deleted.", "name", e.Object.GetName())
				return false
			},
		})).
		Named("application").
		// 额外监听资源，这些资源的变化也会触发调谐
		Owns(&appsv1.Deployment{}, builder.WithPredicates(predicate.Funcs{
			// application 创建的时候会自动创建，不需要这个监听在出发调谐
			CreateFunc: func(e event.CreateEvent) bool {
				return false
			},
			UpdateFunc: func(e event.UpdateEvent) bool {
				// 针对GetResourceVersion字段进行匹配，如果不一致进行调谐
				if e.ObjectNew.GetResourceVersion() == e.ObjectOld.GetResourceVersion() {
					return false
				}
				// 如果新旧spec字段相同也不触发调谐
				// 这里要断言的类型应该是需要监听的类型，不能断言Application
				if reflect.DeepEqual(e.ObjectNew.(*appsv1.Deployment).Spec, e.ObjectOld.(*appsv1.Deployment).Spec) {
					return false
				}
				// 其他情况下进行调谐
				setupLog.Info("The Application Deployment has been Updated.", "name", e.ObjectNew.GetName())
				return true
			},
			// 如果这个deployment被误删除了，应该能主动创建，所以触发调谐
			DeleteFunc: func(e event.DeleteEvent) bool {
				setupLog.Info("The Application Deployment has been Deleted.", "name", e.Object.GetName())
				return true
			},
		})).
		// 额外监听资源，这些资源的变化也会触发调谐
		Owns(&corev1.Service{}, builder.WithPredicates(predicate.Funcs{
			// application 创建的时候会自动创建，不需要这个监听在出发调谐
			CreateFunc: func(e event.CreateEvent) bool {
				return false
			},
			// 更新情况如果
			// 1.ResourceVersion一致不进行更新
			// 2.资源的Spec字段相同不进行更新
			// 其他情况下进行更新
			UpdateFunc: func(e event.UpdateEvent) bool {
				// 针对GetResourceVersion字段进行匹配，如果不一致进行调谐
				if e.ObjectNew.GetResourceVersion() == e.ObjectOld.GetResourceVersion() {
					return false
				}
				// 如果新旧spec字段相同也不触发调谐
				// 这里要断言的类型应该是需要监听的类型，不能断言Application
				if reflect.DeepEqual(e.ObjectNew.(*corev1.Service).Spec, e.ObjectOld.(*corev1.Service).Spec) {
					return false
				}
				// 其他情况下进行调谐
				setupLog.Info("The Application Service has been Updated.", "name", e.ObjectNew.GetName())
				return true
			},
			// 如果这个service被误删除了，应该能主动创建，所以触发调谐
			DeleteFunc: func(e event.DeleteEvent) bool {
				setupLog.Info("The Application Service has been Deleted.", "name", e.Object.GetName())
				return true
			},
		})).
		// MaxConcurrentReconciles 表示控制器同时处理的最大并发调谐（reconciliation）数量，默认是1，就是每次可以支持的最大goroutine数量，这个决定了单位时间内处理事件的能力。和系统资源有关
		WithOptions(controller.Options{MaxConcurrentReconciles: 10}).
		Complete(r)
}
