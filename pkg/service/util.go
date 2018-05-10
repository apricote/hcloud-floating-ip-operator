package service

import (
	hcloudv1alpha1 "github.com/apricote/hcloud-floating-ip-operator/apis/hcloud/v1alpha1"
)

func max(x, y hcloudv1alpha1.Seconds) hcloudv1alpha1.Seconds {
	if x > y {
		return x
	}
	return y
}
