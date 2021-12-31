package charts

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func GetPlacement(cloud *pulumi.StackReference) pulumi.Map {
	return pulumi.Map{
		"nodeSelector": pulumi.Map{
			"beta.kubernetes.io/os":           pulumi.String("linux"),
			"doks.digitalocean.com/node-pool": cloud.GetStringOutput(pulumi.String("toolsNodePoolName")),
		},
		"tolerations": pulumi.Array{
			pulumi.Map{
				"key":      pulumi.String("tools"),
				"operator": pulumi.String("Equal"),
				"value":    pulumi.String("true"),
				"effect":   pulumi.String("NoSchedule"),
			},
		},
	}
}

type CreateChartArgs struct {
	Cloud    *pulumi.StackReference
	Auth     *pulumi.StackReference
	Provider *kubernetes.Provider
}
