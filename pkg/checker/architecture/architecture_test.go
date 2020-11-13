package architecture

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	utils "github.com/jetstack/version-checker/pkg/checker/internal/test"
)

func TestAdd(t *testing.T) {
	tests := map[string]struct {
		input              []*corev1.Node
		expOperationOutput error
		expResult          map[string]*nodeMetadata
	}{
		"update valid node": {
			input: []*corev1.Node{
				utils.CreateNode("node1", utils.ArchAMD64, utils.OSLinux),
				utils.CreateNode("node1", utils.ArchARM, utils.OSLinux),
			},
			expOperationOutput: nil,
			expResult: map[string]*nodeMetadata{
				"node1": &nodeMetadata{
					Architecture: utils.ArchARM,
					OS:           utils.OSLinux,
				},
			},
		},
		"add valid node": {
			input: []*corev1.Node{
				utils.CreateNode("node1", utils.ArchAMD64, utils.OSLinux),
			},
			expOperationOutput: nil,
			expResult: map[string]*nodeMetadata{
				"node1": &nodeMetadata{
					Architecture: utils.ArchAMD64,
					OS:           utils.OSLinux,
				},
			},
		},
		"add nil node": {
			input:              nil,
			expOperationOutput: errors.New("passed node is nil"),
			expResult:          make(map[string]*nodeMetadata),
		},
		"add node with no architecture label": {
			input: []*corev1.Node{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node1",
						Labels: map[string]string{
							corev1.LabelOSStable: "linux",
						},
					},
				},
			},
			expOperationOutput: fmt.Errorf("missing \"kubernetes.io/arch\" label on node \"node1\""),
			expResult:          make(map[string]*nodeMetadata),
		},
		"add node with no os label": {
			input: []*corev1.Node{
				&corev1.Node{
					ObjectMeta: metav1.ObjectMeta{
						Name: "node1",
						Labels: map[string]string{
							corev1.LabelArchStable: "amd64",
						},
					},
				},
			},
			expOperationOutput: fmt.Errorf("missing %q label on node \"node1\"", corev1.LabelOSStable),
			expResult:          make(map[string]*nodeMetadata),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var nodeMap = New()
			for _, node := range test.input {
				err := nodeMap.Add(node)
				if !reflect.DeepEqual(test.expOperationOutput, err) {
					t.Errorf("got unexpected operation error, exp=%s act=%s", test.expOperationOutput, err)
				}
			}
			if !reflect.DeepEqual(nodeMap.nodes, test.expResult) {
				t.Errorf("got unexpected result, exp=%#+v got=%#+v", test.expResult, nodeMap.nodes)
			}

		})
	}
}

func TestDelete(t *testing.T) {
	tests := map[string]struct {
		mapInitialState    []*corev1.Node
		input              string
		expOperationOutput error
		expResult          map[string]*nodeMetadata
	}{
		"delete valid node": {
			mapInitialState: []*corev1.Node{
				utils.CreateNode("node1", utils.ArchAMD64, utils.OSLinux),
				utils.CreateNode("node2", utils.ArchARM, utils.OSLinux),
			},
			input:              "node1",
			expOperationOutput: nil,
			expResult: map[string]*nodeMetadata{
				"node2": &nodeMetadata{
					OS:           utils.OSLinux,
					Architecture: utils.ArchARM,
				},
			},
		},
		"delete empty node name": {
			mapInitialState: []*corev1.Node{
				utils.CreateNode("node1", utils.ArchAMD64, utils.OSLinux),
				utils.CreateNode("node2", utils.ArchARM, utils.OSLinux),
			},
			input:              "",
			expOperationOutput: nil,
			expResult: map[string]*nodeMetadata{
				"node1": &nodeMetadata{
					OS:           utils.OSLinux,
					Architecture: utils.ArchAMD64,
				},
				"node2": &nodeMetadata{
					OS:           utils.OSLinux,
					Architecture: utils.ArchARM,
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var nodeMap = New()
			for _, node := range test.mapInitialState {
				err := nodeMap.Add(node)
				if err != nil {
					t.Errorf("unexpected error: %s", err)
				}
			}

			nodeMap.Delete(test.input)
			if !reflect.DeepEqual(nodeMap.nodes, test.expResult) {
				t.Errorf("got unexpected result, exp=%#+v got=%#+v", test.expResult, nodeMap.nodes)
			}

		})
	}
}
