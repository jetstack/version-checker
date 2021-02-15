package test

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ArchAMD64 = "amd64"
	ArchARM   = "arm/v6"
	OSLinux   = "linux"
)

func CreatePodWithNode(podName, nodeName string) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
		},
		Spec: corev1.PodSpec{
			NodeName: nodeName,
		},
	}
}

func CreatePod(metadata *metav1.ObjectMeta) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: *metadata,
	}
}

func CreateNode(nodeName, arch, os string) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name: nodeName,
			Labels: map[string]string{
				corev1.LabelArchStable: arch,
				corev1.LabelOSStable:   os,
			},
		},
	}
}
