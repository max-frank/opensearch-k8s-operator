package controllers

import (
	"context"
	"fmt"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	opsterv1 "opensearch.opster.io/api/v1"
	"opensearch.opster.io/pkg/helpers"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	//+kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var _ = Describe("OpensearchCluster Controller", func() {
	//	ctx := context.Background()

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		ClusterName       = "cluster-test-dash"
		ClusterNameSpaces = "default"
		timeout           = time.Second * 30
		interval          = time.Second * 1
	)
	var (
		OpensearchCluster = ComposeOpensearchCrd(ClusterName, ClusterNameSpaces)
		cm                = corev1.ConfigMap{}
		//	nodePool          = sts.StatefulSet{}
		service = corev1.Service{}
		deploy  = appsv1.Deployment{}
		//cluster           = opsterv1.OpenSearchCluster{}
		//cluster2 = opsterv1.OpenSearchCluster{}
	)

	/// ------- Creation Check phase -------

	ns := ComposeNs(ClusterName)
	Context("When create OpenSearch CRD - dash", func() {
		It("should create cluster NS", func() {
			Expect(k8sClient.Create(context.Background(), &OpensearchCluster)).Should(Succeed())
			By("Create cluster ns ")
			Eventually(func() bool {

				if !IsNsCreated(k8sClient, ns) {
					return false
				}
				if !IsClusterCreated(k8sClient, OpensearchCluster) {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
		})
	})

	/// ------- Tests logic Check phase -------

	Context("When createing a OpenSearchCluster kind Instance - and Dashboard is Enable", func() {
		It("should create all Opensearch-dashboard resources", func() {
			//fmt.Println(OpensearchCluster)
			fmt.Println("\n DAShBOARD - START")

			By("Opensearch Dashboard")
			Eventually(func() bool {
				fmt.Println("\n DAShBOARD - START - 2")
				//// -------- Dashboard tests ---------
				if err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: ClusterName, Name: ClusterName + "-dashboards"}, &deploy); err != nil {
					return false
				}
				if err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: ClusterName, Name: ClusterName + "-dashboards-config"}, &cm); err != nil {
					return false
				}
				if err := k8sClient.Get(context.Background(), client.ObjectKey{Namespace: ClusterName, Name: OpensearchCluster.Spec.General.ServiceName + "-dashboards"}, &service); err != nil {
					return false
				}
				return true
			}, timeout, interval).Should(BeTrue())
		})
	})

	/// ------- Deletion Check phase -------

	Context("When deleting OpenSearch CRD ", func() {
		It("should delete cluster NS and resources", func() {

			Expect(k8sClient.Delete(context.Background(), &OpensearchCluster)).Should(Succeed())

			By("Delete cluster ns ")
			Eventually(func() bool {
				fmt.Println("\n check ns dashboard")
				return IsNsDeleted(k8sClient, ns)
			}, timeout, interval).Should(BeTrue())
		})
	})

	Context("When running the dashboards reconciler with TLS enabled and an existing cert in a single secret", func() {
		It("should mount the secret", func() {
			clusterName := "dashboards-singlesecret"
			secretName := "my-cert"
			spec := opsterv1.OpenSearchCluster{Spec: opsterv1.ClusterSpec{
				General: opsterv1.GeneralConfig{ClusterName: clusterName, ServiceName: clusterName},
				Dashboards: opsterv1.DashboardsConfig{
					Enable: true,
					Tls: &opsterv1.DashboardsTlsConfig{
						Enable:   true,
						Generate: false,
						Secret:   secretName,
					},
				},
			}}
			ns := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterName,
				},
			}
			fmt.Printf("%s", k8sClient)
			err := k8sClient.Create(context.Background(), &ns)
			Expect(err).ToNot(HaveOccurred())
			underTest := DashboardReconciler{
				Client:   k8sClient,
				Instance: &spec,
				Logger:   logr.Discard(),
				Recorder: &helpers.MockEventRecorder{},
			}
			controllerContext := NewControllerContext()
			_, err = underTest.Reconcile(&controllerContext)
			Expect(err).ToNot(HaveOccurred())
			deployment := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), client.ObjectKey{Name: clusterName + "-dashboards", Namespace: clusterName}, &deployment)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(helpers.CheckVolumeExists(deployment.Spec.Template.Spec.Volumes, deployment.Spec.Template.Spec.Containers[0].VolumeMounts, secretName, "tls-cert")).Should((BeTrue()))
		})
	})

	Context("When running the dashboards reconciler with TLS enabled and an existing cert/key in separate secrets", func() {
		It("should mount the secrets", func() {
			clusterName := "dashboards-test-multisecret"
			keySecretName := "my-key"
			certSecretName := "my-cert"
			spec := opsterv1.OpenSearchCluster{Spec: opsterv1.ClusterSpec{
				General: opsterv1.GeneralConfig{ClusterName: clusterName, ServiceName: clusterName},
				Dashboards: opsterv1.DashboardsConfig{
					Enable: true,
					Tls: &opsterv1.DashboardsTlsConfig{
						Enable:     true,
						Generate:   false,
						KeySecret:  &opsterv1.TlsSecret{SecretName: keySecretName},
						CertSecret: &opsterv1.TlsSecret{SecretName: certSecretName},
					},
				},
			}}
			ns := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterName,
				},
			}
			fmt.Printf("%s", k8sClient)
			err := k8sClient.Create(context.Background(), &ns)
			Expect(err).ToNot(HaveOccurred())
			underTest := DashboardReconciler{
				Client:   k8sClient,
				Instance: &spec,
				Logger:   logr.Discard(),
				Recorder: &helpers.MockEventRecorder{},
			}
			controllerContext := NewControllerContext()
			_, err = underTest.Reconcile(&controllerContext)
			Expect(err).ToNot(HaveOccurred())
			deployment := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), client.ObjectKey{Name: clusterName + "-dashboards", Namespace: clusterName}, &deployment)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(helpers.CheckVolumeExists(deployment.Spec.Template.Spec.Volumes, deployment.Spec.Template.Spec.Containers[0].VolumeMounts, keySecretName, "tls-key")).Should((BeTrue()))
			Expect(helpers.CheckVolumeExists(deployment.Spec.Template.Spec.Volumes, deployment.Spec.Template.Spec.Containers[0].VolumeMounts, certSecretName, "tls-cert")).Should((BeTrue()))
		})
	})

	Context("When running the dashboards reconciler with TLS enabled and generate enabled", func() {
		It("should create a cert", func() {
			clusterName := "dashboards-test-generate"
			spec := opsterv1.OpenSearchCluster{Spec: opsterv1.ClusterSpec{
				General: opsterv1.GeneralConfig{ClusterName: clusterName, ServiceName: clusterName},
				Dashboards: opsterv1.DashboardsConfig{
					Enable: true,
					Tls: &opsterv1.DashboardsTlsConfig{
						Enable:   true,
						Generate: true,
					},
				},
			}}
			ns := corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: clusterName,
				},
			}
			fmt.Printf("%s", k8sClient)
			err := k8sClient.Create(context.Background(), &ns)
			Expect(err).ToNot(HaveOccurred())
			underTest := DashboardReconciler{
				Client:   k8sClient,
				Instance: &spec,
				Logger:   logr.Discard(),
				Recorder: &helpers.MockEventRecorder{},
			}
			controllerContext := NewControllerContext()
			_, err = underTest.Reconcile(&controllerContext)
			Expect(err).ToNot(HaveOccurred())
			// Check if secret is mounted
			deployment := appsv1.Deployment{}
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), client.ObjectKey{Name: clusterName + "-dashboards", Namespace: clusterName}, &deployment)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(helpers.CheckVolumeExists(deployment.Spec.Template.Spec.Volumes, deployment.Spec.Template.Spec.Containers[0].VolumeMounts, clusterName+"-dashboards-cert", "tls-cert")).Should((BeTrue()))
			// Check if secret contains correct data keys
			secret := corev1.Secret{}
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), client.ObjectKey{Name: clusterName + "-dashboards-cert", Namespace: clusterName}, &secret)
				return err == nil
			}, timeout, interval).Should(BeTrue())
			Expect(helpers.HasKeyWithBytes(secret.Data, "tls.key")).To(BeTrue())
			Expect(helpers.HasKeyWithBytes(secret.Data, "tls.crt")).To(BeTrue())
		})
	})

})
