package controller

import (
	"context"
	"reflect"

	appv1 "github.com/aloys.zy/aloys-application-operator-webhook/api/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (r *ApplicationReconciler) reconcileService(ctx context.Context, application *appv1.Application) (ctrl.Result, error) {
	// TODO: Implement the logic to reconcile the service
	// This can involve checking the current state of the service, comparing it with the desired state,
	// and applying necessary changes to achieve the desired state.
	// You can use Kubernetes client-go to interact with the Kubernetes API server.
	// controllerutil.CreateOrPatch() 或者	// ctrl.CreateOrUpdate() 替换手工判断创建
	setupLog := log.FromContext(ctx).WithName("reconcileService")
	appNamespace := application.Namespace
	appName := application.Name + "-service"
	svc := &corev1.Service{}
	// err := r.Get(ctx, types.NamespacedName{Namespace: application.Namespace, Name: application.Name}, dp)
	// client.ObjectKey 就是types.NamespacedName的别名,写法是等价的
	// controllerutil.CreateOrUpdate：会比较整个对象（包括 spec 和 status 字段），如果任何一个字段发生变化，都会触发全量更新操作。
	// controllerutil.CreateOrPatch：会分别比较 spec 和 status 字段，如果 spec 字段发生变化，仅更新 spec；如果 status 字段发生变化，仅更新 status
	// func() error在进行创建和更新之前都先调用回调函数进行修改
	// 直接使用 controllerutil.CreateOrPatch和controllerutil.CreateOrUpdate 感觉更麻烦，需要多次请求api
	err := r.Get(ctx, client.ObjectKey{Namespace: appNamespace, Name: appName}, svc)
	// 如果service资源存在
	if err == nil {
		setupLog.Info("The service has already exist.", "ServiceNamespace", appNamespace, "ServiceName", appName)
		// 判断application.Status是不是最新的,最新的就结束本次调谐
		if reflect.DeepEqual(svc.Status, application.Status.Network) {
			// setupLog.Info("The service status is already the same as the desired status.", "ServiceNamespace", appNamespace, "ServiceName", appName)
			return ctrl.Result{}, nil
		}
		// 不是最新的就进行赋值，更新状态,更新失败进行重试
		application.Status.Network = svc.Status
		if err := r.Status().Update(ctx, application); err != nil {
			setupLog.Error(err, "Failed to update the service status.", "ServiceNamespace", appNamespace, "ServiceName", appName)
			return ctrl.Result{RequeueAfter: GenericRequeueDuration}, err
		}
		setupLog.Info("The Application Network status has been updated.", "name", appNamespace, "name", appName)
		return ctrl.Result{}, nil
	}
	// 先进行判断是不是不存在之外的错误场景，不是不存的错误就直接重试
	if !errors.IsNotFound(err) {
		setupLog.Error(err, "Failed to get the Service,will request after a short time.", "ServiceNamespace", appNamespace, "ServiceName", appName)
		return ctrl.Result{RequeueAfter: GenericRequeueDuration}, err
	}
	svc.SetName(appName)
	svc.SetNamespace(appNamespace)
	svc.SetLabels(application.Labels)
	svc.Spec = application.Spec.Service.ServiceSpec
	svc.Spec.Selector = application.Labels
	// 设置 OwnerReference，使 svc 成为 Application 的子资源
	if err := ctrl.SetControllerReference(application, svc, r.Scheme); err != nil {
		setupLog.Error(err, "Failed to set the owner reference for the Service.", "ServiceNamespace", appNamespace, "ServiceName", appName)
		return ctrl.Result{RequeueAfter: GenericRequeueDuration}, err
	}
	// 状态和应用状态进行关联，这里是没有必要的创建 Service 后，Kubernetes 会自动触发事件，这些事件会被控制器捕获，控制器会重新调用 Reconcile 函数，此时可以检查并更新 application 的状态，等待事件触发，让 Reconcile 函数自然地处理状态更新，不需要重复的触发Reconcile。 application.Status.Workflow = dp.Status 创建后不需要再次触发更新application.status
	if err := r.Create(ctx, svc); err != nil {
		setupLog.Error(err, "Failed to create the Service.", "ServiceNamespace", appNamespace, "ServiceName", appName)
		return ctrl.Result{RequeueAfter: GenericRequeueDuration}, err
	}
	setupLog.Info("The Service has been created.", "ServiceNamespace", appNamespace, "ServiceName", appName)
	return ctrl.Result{}, nil
}
