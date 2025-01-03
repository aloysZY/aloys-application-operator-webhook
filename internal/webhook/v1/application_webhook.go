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

package v1

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	appsv1 "github.com/aloys.zy/aloys-application-operator-webhook/api/v1"
)

// nolint:unused
// log is for logging in this package.
var applicationlog = logf.Log.WithName("application-resource")

// SetupApplicationWebhookWithManager registers the webhook for Application in the manager.
func SetupApplicationWebhookWithManager(mgr ctrl.Manager) error {
	// 使用 NewWebhookManagedBy 方法创建一个新的 webhook，并设置了验证器和默认值处理器
	return ctrl.NewWebhookManagedBy(mgr).For(&appsv1.Application{}).
		// WithValidator数据验证
		WithValidator(&ApplicationCustomValidator{}).
		// WithDefaulter数据修改
		// 自定义字段初始化后再校验 ApplicationCustomDefaulter这个实例随后被注册到 webhook 中，以确保每当一个新的 Application 资源被创建或更新时，都会调用这个 defaulter 来设置默认值
		WithDefaulter(&ApplicationCustomDefaulter{DefaultReplicas: 1}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-apps-aloys-cn-v1-application,mutating=true,failurePolicy=fail,sideEffects=None,groups=apps.aloys.cn,resources=applications,verbs=create;update,versions=v1,name=mapplication-v1.kb.io,admissionReviewVersions=v1

// ApplicationCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Application when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type ApplicationCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting

	// 可以自定义一些字段内容，在Default内进行使用
	DefaultReplicas int32 `json:"-"`
	// DefaultImage    string `json:"-"`
}

// 确保ApplicationCustomDefaulter 结构体实现了 CustomDefaulter 接口
var _ webhook.CustomDefaulter = &ApplicationCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Application.
func (d *ApplicationCustomDefaulter) Default(ctx context.Context, obj runtime.Object) error {
	application, ok := obj.(*appsv1.Application)

	if !ok {
		return fmt.Errorf("expected an Application object but got %T", obj)
	}
	applicationlog.Info("Defaulting for Application", "name", application.GetName())

	// TODO(user): fill in your defaulting logic.
	// 设置默认副本数量
	if application.Spec.Deployment.Replicas == nil {
		application.Spec.Deployment.Replicas = &d.DefaultReplicas
	}
	// 判断副本数量，并限制最大值为 8
	if *application.Spec.Deployment.Replicas > 9 {
		maxReplicas := int32(8)                             // 创建一个 int32 类型的变量
		application.Spec.Deployment.Replicas = &maxReplicas // 将其地址赋值给 Replicas
		applicationlog.Info("Setting default replicas for application.", "ApplicationName", application.Name, "NewReplicas", d.DefaultReplicas)
	}
	// // 追加标签
	// labels := make(map[string]string)
	// labels["app"] = application.Name + "-----xxx"
	// application.Spec.Deployment.Template.SetLabels(labels)
	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-apps-aloys-cn-v1-application,mutating=false,failurePolicy=fail,sideEffects=None,groups=apps.aloys.cn,resources=applications,verbs=create;update,versions=v1,name=vapplication-v1.kb.io,admissionReviewVersions=v1

// ApplicationCustomValidator struct is responsible for validating the Application resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type ApplicationCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

// 确保ApplicationCustomValidator 结构体实现了 CustomValidator 接口
var _ webhook.CustomValidator = &ApplicationCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Application.
func (v *ApplicationCustomValidator) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	application, ok := obj.(*appsv1.Application)
	if !ok {
		return nil, fmt.Errorf("expected a Application object but got %T", obj)
	}
	applicationlog.Info("Validation for Application upon creation", "name", application.GetName())

	// TODO(user): fill in your validation logic upon object creation.

	return nil, nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Application.
func (v *ApplicationCustomValidator) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	application, ok := newObj.(*appsv1.Application)
	if !ok {
		return nil, fmt.Errorf("expected a Application object for the newObj but got %T", newObj)
	}
	applicationlog.Info("Validation for Application upon update", "name", application.GetName())

	// TODO(user): fill in your validation logic upon object update.

	return nil, nil
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Application.
func (v *ApplicationCustomValidator) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	application, ok := obj.(*appsv1.Application)
	if !ok {
		return nil, fmt.Errorf("expected a Application object but got %T", obj)
	}
	applicationlog.Info("Validation for Application upon deletion", "name", application.GetName())

	// TODO(user): fill in your validation logic upon object deletion.

	return nil, nil
}
