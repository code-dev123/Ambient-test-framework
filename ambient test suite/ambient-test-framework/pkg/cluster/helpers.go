// pkg/cluster/helpers.go
package cluster

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func listOptsWithContext(_ context.Context) metav1.ListOptions {
	return metav1.ListOptions{}
}

func getOptsWithContext(_ context.Context) metav1.GetOptions {
	return metav1.GetOptions{}
}
