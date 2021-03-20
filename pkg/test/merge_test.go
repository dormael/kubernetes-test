package test

import (
	"fmt"
	"testing"

	"github.com/imdario/mergo"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/json"
)

func TestMergeUnstructured(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Unstructured Suite")

}

var _ = Describe("Unstructured", func() {
	Context("Merge should work", func() {

		It("For nested values", func() {
			name := "dst"

			src := emptyUnstructured()
			dst := emptyUnstructured()

			src.SetAPIVersion("beta1")
			src.SetClusterName("cluster")

			dst.SetAPIVersion("alpha1")
			dst.SetName(name)

			err := mergo.Map(&dst.Object, src.Object, mergo.WithOverride)
			Expect(err).Should(BeNil())

			Expect(dst.GetAPIVersion()).Should(Equal(src.GetAPIVersion()))
			Expect(dst.GetClusterName()).Should(Equal(src.GetClusterName()))
			Expect(dst.GetName()).Should(Equal(name))

			fmt.Println(dst)
		})

		It("For nested ports", func() {
			srcPort := int32(8080)
			dstPort := srcPort + 1

			src := newDeploymentWithPort(srcPort)
			dst := newDeploymentWithPort(dstPort)

			err := mergo.Merge(dst, src, mergo.WithOverride)
			Expect(err).Should(BeNil())

			_containers := dst.Spec.Template.Spec.Containers
			Expect(len(_containers)).Should(Equal(1))

			_ports := _containers[0].Ports
			Expect(len(_ports)).Should(Equal(1))
			Expect(_ports[0].HostPort).Should(BeNumerically("==", srcPort))
			Expect(_ports[0].ContainerPort).Should(BeNumerically("==", srcPort))
		})

		It("For nested ports(unstructured)", func() {
			srcPort := int32(8080)
			dstPort := srcPort + 1

			src := newDeploymentWithPort(srcPort)
			dst := newDeploymentWithPort(dstPort)

			testForUnstructured(src, dst)
		})

		It("For nested multiple ports", func() {
			srcPort := int32(8080)
			src := newDeploymentWithContainers([]corev1.Container{
				{
					Ports: []corev1.ContainerPort{
						newPort(srcPort),
						newPort(srcPort + 1),
					},
				},
			})
			dst := newDeploymentWithContainers([]corev1.Container{
				{
					Ports: []corev1.ContainerPort{
						newPort(srcPort + 1),
						newPort(srcPort + 2),
					},
				},
			})

			err := mergo.Merge(dst, src, mergo.WithOverride)
			Expect(err).Should(BeNil())

			containers := dst.Spec.Template.Spec.Containers
			Expect(len(containers)).Should(Equal(1))

			ports := containers[0].Ports
			Expect(len(ports)).Should(Equal(2))
			Expect(ports[0].HostPort).Should(BeNumerically("==", srcPort))
			Expect(ports[0].ContainerPort).Should(BeNumerically("==", srcPort))
			Expect(ports[1].HostPort).Should(BeNumerically("==", srcPort+1))
			Expect(ports[1].ContainerPort).Should(BeNumerically("==", srcPort+1))
		})

		It("For nested multiple ports(unstructured)", func() {
			srcPort := int32(8080)

			src := newDeploymentWithContainers([]corev1.Container{
				{
					Ports: []corev1.ContainerPort{
						newPort(srcPort + 1),
						newPort(srcPort),
					},
				},
			})
			dst := newDeploymentWithContainers([]corev1.Container{
				{
					Ports: []corev1.ContainerPort{
						newPort(srcPort),
						newPort(srcPort + 1),
						newPort(srcPort + 2),
					},
				},
			})

			testForUnstructured(src, dst)
		})
	})
})

func testForUnstructured(src *appsv1.Deployment, dst *appsv1.Deployment) {
	_containers := src.Spec.Template.Spec.Containers
	_ports := _containers[0].Ports

	srcBytes, srcError := json.Marshal(src)
	Expect(srcError).Should(BeNil())

	dstBytes, dstError := json.Marshal(dst)
	Expect(dstError).Should(BeNil())

	srcUnstructured := emptyUnstructured()
	dstUnstructured := emptyUnstructured()

	err := json.Unmarshal(srcBytes, &srcUnstructured)
	Expect(err).Should(BeNil())

	err = json.Unmarshal(dstBytes, &dstUnstructured)
	Expect(err).Should(BeNil())

	err = mergo.Map(&dstUnstructured.Object, srcUnstructured.Object, mergo.WithOverride)
	Expect(err).Should(BeNil())

	containers, ok, errSlice := unstructured.NestedSlice(dstUnstructured.Object, "spec", "template", "spec", "containers")
	Expect(errSlice).Should(BeNil())
	Expect(ok).Should(Equal(true))
	Expect(len(containers)).Should(Equal(1))

	containerMap := containers[0].(map[string]interface{})

	ports, ok, errSlice := unstructured.NestedSlice(containerMap, "ports")
	Expect(errSlice).Should(BeNil())
	Expect(ok).Should(Equal(true))
	Expect(len(ports)).Should(Equal(len(_ports)))

	for idx := range ports {
		portMap := ports[idx].(map[string]interface{})

		hostPort, ok, errInt := unstructured.NestedInt64(portMap, "hostPort")
		Expect(errInt).Should(BeNil())
		Expect(ok).Should(Equal(true))
		Expect(hostPort).Should(BeNumerically("==", _ports[idx].HostPort))

		containerPort, ok, errInt := unstructured.NestedInt64(portMap, "containerPort")
		Expect(errInt).Should(BeNil())
		Expect(ok).Should(Equal(true))
		Expect(containerPort).Should(BeNumerically("==", _ports[idx].ContainerPort))
	}
}

func emptyUnstructured() unstructured.Unstructured {
	return unstructured.Unstructured{Object: make(map[string]interface{}, 0)}
}

func newDeploymentWithPort(port int32) *appsv1.Deployment {
	containers := []corev1.Container{
		{
			Ports: []corev1.ContainerPort{
				newPort(port),
			},
		},
	}
	return newDeploymentWithContainers(containers)
}

func newDeploymentWithContainers(containers []corev1.Container) *appsv1.Deployment {
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "deployment01",
		},
		Spec: appsv1.DeploymentSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: containers,
				},
			},
		},
	}
}

func newPort(port int32) corev1.ContainerPort {
	return corev1.ContainerPort{
		Name:          fmt.Sprintf("port-%d", port),
		ContainerPort: port,
		HostPort:      port,
		Protocol:      corev1.ProtocolTCP,
	}
}
