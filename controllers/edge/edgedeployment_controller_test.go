package edge

import (
	"context"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	edgev1alpha1 "github.com/pbowden/edge-deploy/apis/edge/v1alpha1"
)

var _ = Describe("EdgeDeployment controller logic", func() {

	// Define utility constants for object names and testing timeouts/durations and intervals.
	const (
		EdgeDeploymentName      = "test-edgedeployment-1"
		EdgeDeploymentNamespace = "default"
		JobName                 = "test-job"

		timeout  = time.Second * 10
		duration = time.Second * 10
		interval = time.Millisecond * 250
	)

	Context("When creating EdgeDeployment spec", func() {
		It("Should create PodSpecs", func() {
			By("By creating a new EdgeDeployment")
			ctx := context.Background()
			edgeDeployment := &edgev1alpha1.EdgeDeployment{
				ObjectMeta: v1.ObjectMeta{
					Name:      EdgeDeploymentName,
					Namespace: EdgeDeploymentNamespace,
				},
				Spec: edgev1alpha1.EdgeDeploymentSpec{
					EdgeNodes: []string{"testedgenode1", "testedgenode2", "testedgenode5"},
					Template: edgev1alpha1.EdgePodTemplateSpec{
						Spec: edgev1alpha1.EdgePodSpec{
							Containers: []edgev1alpha1.EdgeContainer{
								{
									Name:  "nginx",
									Image: "someimage",
									Ports: []edgev1alpha1.ContainerPort{
										{ContainerPort: 8022},
									},
								},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, edgeDeployment)).Should(Succeed())

			foundEdgeDeployment := &edgev1alpha1.EdgeDeployment{}
			edgeDeploymentLookupKey := types.NamespacedName{
				Name:      EdgeDeploymentName,
				Namespace: EdgeDeploymentNamespace,
			}

			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), edgeDeploymentLookupKey, foundEdgeDeployment)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			By("By Creating Three EdgePods")
			foundEdgePodList := &edgev1alpha1.EdgePodList{}
			Eventually(func() ([]string, error) {
				/* 		foundEdgePodList := &edgev1alpha1.EdgePodList{}
				listOpts := []client.ListOption{
					client.InNamespace(edgeDeployment.Namespace),
					client.MatchingLabels(map[string]string{"deploymentName": edgeDeployment.Name}),
				}
				err := k8sClient.List(ctx, foundEdgePodList, listOpts...)
				if err != nil {
					return nil, err
				} */

				getEdgePods(foundEdgePodList, edgeDeployment.Namespace, edgeDeployment.Name)
				edgePodsNames := []string{}
				for _, edgePod := range foundEdgePodList.Items {
					edgePodsNames = append(edgePodsNames, edgePod.Name)
				}
				return edgePodsNames, nil
			}, timeout, interval).Should(ContainElements("testedgenode1-"+EdgeDeploymentName, "testedgenode2-"+EdgeDeploymentName))
			Expect(len(foundEdgePodList.Items)).To(Equal(3))
		})

		It("Should create/update/delete new PodSpecs", func() {
			foundEdgeDeployment := &edgev1alpha1.EdgeDeployment{}
			edgeDeploymentLookupKey := types.NamespacedName{
				Name:      EdgeDeploymentName,
				Namespace: EdgeDeploymentNamespace,
			}

			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), edgeDeploymentLookupKey, foundEdgeDeployment)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			foundEdgeDeployment.Spec.EdgeNodes = []string{"testedgenode2", "testedgenode3", "testedgenode4"}
			foundEdgeDeployment.Spec.Template.Spec.Containers = []edgev1alpha1.EdgeContainer{
				{
					Name:  "nginx",
					Image: "someimage",
					Ports: []edgev1alpha1.ContainerPort{
						{ContainerPort: 8023},
					},
				},
			}

			Expect(k8sClient.Update(context.Background(), foundEdgeDeployment)).Should(Succeed())
			time.Sleep(time.Second * 1)
			// Fetch updated EdgeDeployment
			Eventually(func() bool {
				err := k8sClient.Get(context.Background(), edgeDeploymentLookupKey, foundEdgeDeployment)
				return err == nil
			}, timeout, interval).Should(BeTrue())

			foundEdgePodList := &edgev1alpha1.EdgePodList{}
			getEdgePods(foundEdgePodList, foundEdgeDeployment.Namespace, foundEdgeDeployment.Name)

			By("Creating the proper number of Podspecs")
			Expect(len(foundEdgePodList.Items)).To(Equal(3))

			By("Updating any existing Podspecs in deployment that have been modified", func() {
				//	testedgenode2 := &edgev1alpha1.EdgePod{}
				edgePodName := "testedgenode2"
				foundEdgePod := &edgev1alpha1.EdgePod{}
				for _, edgePod := range foundEdgePodList.Items {
					if edgePod.Name == edgePodName+"-"+foundEdgeDeployment.Name {
						foundEdgePod = &edgePod
						break
					}

				}
				Expect(foundEdgePod.Spec).To(Equal(foundEdgeDeployment.Spec.Template.Spec))
			})

			By("Removing PodSpecs no longer in deployment", func() {
				Expect(foundEdgePodList.Items).NotTo(ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"EdgeTarget": Or(Equal("testedgenode1"), Equal("testedegenode5")),
				})))

			})

			By("Creating any new PodSpecs in deployment", func() {
				Expect(foundEdgePodList.Items).To(ContainElement(gstruct.MatchFields(gstruct.IgnoreExtras, gstruct.Fields{
					"EdgeTarget": Or(Equal("testedgenode2"), Equal("testedgenode3"), Equal("testedgenode4")),
				})))
			})
		})

	})
})

func getEdgePods(podlist *edgev1alpha1.EdgePodList, edgeDeploymentNamespace string, edgeDeploymentName string) error {
	//foundEdgePodList := &edgev1alpha1.EdgePodList{}
	listOpts := []client.ListOption{
		client.InNamespace(edgeDeploymentNamespace),
		client.MatchingLabels(map[string]string{"deploymentName": edgeDeploymentName}),
	}
	err := k8sClient.List(context.Background(), podlist, listOpts...)
	if err != nil {
		return err
	}
	return nil
}
