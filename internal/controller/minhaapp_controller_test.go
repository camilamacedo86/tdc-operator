/*
Copyright 2025.

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
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	exemplocomv1alpha1 "github/tdc-operator/api/v1alpha1"
)

var _ = Describe("MinhaApp controller", func() {
	Context("MinhaApp controller test", func() {

		const MinhaAppName = "test-minhaapp"

		ctx := context.Background()

		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name:      MinhaAppName,
				Namespace: MinhaAppName,
			},
		}

		typeNamespacedName := types.NamespacedName{
			Name:      MinhaAppName,
			Namespace: MinhaAppName,
		}
		minhaapp := &exemplocomv1alpha1.MinhaApp{}

		SetDefaultEventuallyTimeout(2 * time.Minute)
		SetDefaultEventuallyPollingInterval(time.Second)

		BeforeEach(func() {
			By("Creating the Namespace to perform the tests")
			err := k8sClient.Create(ctx, namespace)
			Expect(err).NotTo(HaveOccurred())

			By("Setting the Image ENV VAR which stores the Operand image")
			err = os.Setenv("MINHAAPP_IMAGE", "example.com/image:test")
			Expect(err).NotTo(HaveOccurred())

			By("creating the custom resource for the Kind MinhaApp")
			err = k8sClient.Get(ctx, typeNamespacedName, minhaapp)
			if err != nil && errors.IsNotFound(err) {
				// Let's mock our custom resource at the same way that we would
				// apply on the cluster the manifest under config/samples
				minhaapp = &exemplocomv1alpha1.MinhaApp{
					ObjectMeta: metav1.ObjectMeta{
						Name:      MinhaAppName,
						Namespace: namespace.Name,
					},
					Spec: exemplocomv1alpha1.MinhaAppSpec{
						Size:          ptr.To(int32(1)),
						ContainerPort: 80,
					},
				}

				err = k8sClient.Create(ctx, minhaapp)
				Expect(err).NotTo(HaveOccurred())
			}
		})

		AfterEach(func() {
			By("removing the custom resource for the Kind MinhaApp")
			found := &exemplocomv1alpha1.MinhaApp{}
			err := k8sClient.Get(ctx, typeNamespacedName, found)
			Expect(err).NotTo(HaveOccurred())

			Eventually(func(g Gomega) {
				g.Expect(k8sClient.Delete(context.TODO(), found)).To(Succeed())
			}).Should(Succeed())

			// TODO(user): Attention if you improve this code by adding other context test you MUST
			// be aware of the current delete namespace limitations.
			// More info: https://book.kubebuilder.io/reference/envtest.html#testing-considerations
			By("Deleting the Namespace to perform the tests")
			_ = k8sClient.Delete(ctx, namespace)

			By("Removing the Image ENV VAR which stores the Operand image")
			_ = os.Unsetenv("MINHAAPP_IMAGE")
		})

		It("should successfully reconcile a custom resource for MinhaApp", func() {
			By("Checking if the custom resource was successfully created")
			Eventually(func(g Gomega) {
				found := &exemplocomv1alpha1.MinhaApp{}
				Expect(k8sClient.Get(ctx, typeNamespacedName, found)).To(Succeed())
			}).Should(Succeed())

			By("Reconciling the custom resource created")
			minhaappReconciler := &MinhaAppReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := minhaappReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking if Deployment was successfully created in the reconciliation")
			Eventually(func(g Gomega) {
				found := &appsv1.Deployment{}
				g.Expect(k8sClient.Get(ctx, typeNamespacedName, found)).To(Succeed())
			}).Should(Succeed())

			By("Reconciling the custom resource again")
			_, err = minhaappReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			By("Checking the latest Status Condition added to the MinhaApp instance")
			Expect(k8sClient.Get(ctx, typeNamespacedName, minhaapp)).To(Succeed())
			var conditions []metav1.Condition
			Expect(minhaapp.Status.Conditions).To(ContainElement(
				HaveField("Type", Equal(typeAvailableMinhaApp)), &conditions))
			Expect(conditions).To(HaveLen(1), "Multiple conditions of type %s", typeAvailableMinhaApp)
			Expect(conditions[0].Status).To(Equal(metav1.ConditionTrue), "condition %s", typeAvailableMinhaApp)
			Expect(conditions[0].Reason).To(Equal("Reconciling"), "condition %s", typeAvailableMinhaApp)
		})
	})
})
