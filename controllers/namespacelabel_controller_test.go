package controllers

import (
	"context"
	"fmt"
	"reflect"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	omerv1 "omer.io/namespacelabel/api/v1"
)

var _ = Describe("NamespaceLabel controller", func() {

	const (
		timeout             = "25s"
		interval            = "5s"
		namespacelabelName1 = "a"
		namespacelabelName2 = "b"
		namespacelabelName3 = "c"
		namespace           = "default"
	)

	Context("When setting up the test environment", func() {
		It("Should create NamespaceLabel custom resources", func() {
			By("Creating a first NamespaceLabel custom resource")
			ctx := context.Background()
			nsLabel1 := omerv1.NamespaceLabel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namespacelabelName1,
					Namespace: namespace,
				},
				Spec: omerv1.NamespaceLabelSpec{
					Labels: map[string]string{
						"a": "a",
						"b": "b",
					},
				},
			}
			Expect(k8sClient.Create(ctx, &nsLabel1)).Should(Succeed())

			By("Creating another NamespaceLabel custom resource")
			nsLabel2 := omerv1.NamespaceLabel{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namespacelabelName2,
					Namespace: namespace,
				},
				Spec: omerv1.NamespaceLabelSpec{
					Labels: map[string]string{
						"c": "c",
						"d": "d",
					},
				},
			}
			Expect(k8sClient.Create(ctx, &nsLabel2)).Should(Succeed())
		})
	})

	Context("When creating two namespacelabel custom resources", func() {
		It("Should sync the labels in the nslabels cr with the namespace's labels", func() {
			time.Sleep(time.Second * 2)
			By("Check if the labels in the nslabels is sync with the namespace")
			var namespaceObj v1.Namespace
			namespacedName := types.NamespacedName{Name: namespace}
			Eventually(func() bool {
				if err := k8sClient.Get(ctx, namespacedName, &namespaceObj); err != nil {
					return false
				}
				return reflect.DeepEqual(map[string]string{
					"a":                           "a",
					"b":                           "b",
					"c":                           "c",
					"d":                           "d",
					"kubernetes.io/metadata.name": "default"}, namespaceObj.GetLabels())
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("When change one of the namespace labels that exist in the namespacelabel spec", func() {
		It("Should resync, and update the label back to the label in the nslabel cr", func() {
			time.Sleep(time.Second * 2)
			By("change the value of one of the labels in the namespace")
			var namespaceObj v1.Namespace
			namespacedName := types.NamespacedName{Name: namespace}
			k8sClient.Get(ctx, namespacedName, &namespaceObj)
			namespaceObj.ObjectMeta.Labels["a"] = "hara"
			k8sClient.Update(ctx, &namespaceObj)
			Eventually(func() bool {
				k8sClient.Get(ctx, namespacedName, &namespaceObj)
				return reflect.DeepEqual(map[string]string{
					"a":                           "a",
					"b":                           "b",
					"c":                           "c",
					"d":                           "d",
					"kubernetes.io/metadata.name": "default"}, namespaceObj.GetLabels())
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("When delete one of the labels of nslabel object", func() {
		It("Should resync, and delete the label from the cr and the namespace", func() {
			time.Sleep(time.Second * 2)
			By("delete the value from one of the labels in the nslabel cr")
			var nslabel1 omerv1.NamespaceLabel
			namespacedName := types.NamespacedName{Name: namespacelabelName1, Namespace: namespace}
			k8sClient.Get(ctx, namespacedName, &nslabel1)
			delete(nslabel1.Spec.Labels, "a")
			k8sClient.Update(ctx, &nslabel1)
			Eventually(func() bool {
				var namespaceObj v1.Namespace
				namespacedName = types.NamespacedName{Name: namespace}
				k8sClient.Get(ctx, namespacedName, &namespaceObj)
				namespacedName := types.NamespacedName{Name: namespacelabelName1, Namespace: namespace}
				k8sClient.Get(ctx, namespacedName, &nslabel1)
				fmt.Println(nslabel1.Status.SyncLabels)
				return (reflect.DeepEqual(map[string]string{
					"b":                           "b",
					"c":                           "c",
					"d":                           "d",
					"kubernetes.io/metadata.name": "default"}, namespaceObj.GetLabels())) &&
					(reflect.DeepEqual(map[string]string{
						"b": "b"}, nslabel1.Status.SyncLabels))
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("When add label that already exists in the namespace", func() {
		It("Should resync, and delete the label from the cr", func() {
			time.Sleep(time.Second * 2)
			By("add label to the nslabel cr that already exist in the ns")
			var nslabel1 omerv1.NamespaceLabel
			namespacedName := types.NamespacedName{Name: namespacelabelName1, Namespace: namespace}
			k8sClient.Get(ctx, namespacedName, &nslabel1)
			nslabel1.Spec.Labels["kubernetes.io/metadata.name"] = "haragadol"
			k8sClient.Update(ctx, &nslabel1)
			Eventually(func() bool {
				k8sClient.Get(ctx, namespacedName, &nslabel1)
				return reflect.DeepEqual(map[string]string{
					"b": "b",
				}, nslabel1.Spec.Labels)
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("When add new label to the nslabelobject", func() {
		It("Should resync, and add the label to the ns", func() {
			time.Sleep(time.Second * 2)
			By("add new label to the nslabel cr")
			var nslabel1 omerv1.NamespaceLabel
			namespacedName := types.NamespacedName{Name: namespacelabelName1, Namespace: namespace}
			k8sClient.Get(ctx, namespacedName, &nslabel1)
			nslabel1.Spec.Labels["m"] = "m"
			k8sClient.Update(ctx, &nslabel1)
			Eventually(func() bool {
				var namespaceObj v1.Namespace
				namespacedName = types.NamespacedName{Name: namespace}
				k8sClient.Get(ctx, namespacedName, &namespaceObj)

				return reflect.DeepEqual(map[string]string{
					"b":                           "b",
					"c":                           "c",
					"d":                           "d",
					"kubernetes.io/metadata.name": "default",
					"m":                           "m",
				}, namespaceObj.ObjectMeta.Labels)
			}, timeout, interval).Should(BeTrue())
		})
	})
	Context("When delete nslabel object", func() {
		It("Should resync, and remove the sync labels from the ns", func() {
			time.Sleep(time.Second * 2)
			By("delete nslabel cr")
			var nslabel1 omerv1.NamespaceLabel
			namespacedName := types.NamespacedName{Name: namespacelabelName1, Namespace: namespace}
			k8sClient.Get(ctx, namespacedName, &nslabel1)
			k8sClient.Delete(ctx, &nslabel1)
			Eventually(func() bool {
				var namespaceObj v1.Namespace
				namespacedName = types.NamespacedName{Name: namespace}
				k8sClient.Get(ctx, namespacedName, &namespaceObj)

				return reflect.DeepEqual(map[string]string{
					"c":                           "c",
					"d":                           "d",
					"kubernetes.io/metadata.name": "default",
				}, namespaceObj.ObjectMeta.Labels)
			}, timeout, interval).Should(BeTrue())
		})
	})
})
