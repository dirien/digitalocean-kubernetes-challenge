package charts

import (
	"github.com/pulumi/pulumi-kubernetes/sdk/v3/go/kubernetes/helm/v3"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
)

func NewCole(ctx *pulumi.Context, args CreateChartArgs, parent *helm.Release) (*helm.Release, error) {

	cole, err := helm.NewRelease(ctx, "cole", &helm.ReleaseArgs{
		Chart:     pulumi.String("cole"),
		Version:   pulumi.String("1.0.1"),
		Namespace: pulumi.String("monitoring"),
		RepositoryOpts: helm.RepositoryOptsArgs{
			Repo: pulumi.String("https://nicolastakashi.github.io/cole"),
		},
		Values: pulumi.Map{
			"serviceMonitor": pulumi.Map{
				"enabled": pulumi.Bool(true),
			},
			"flags": pulumi.Map{
				"grafana": pulumi.Map{
					"namespace": pulumi.String("monitoring"),
				},
			},
		},
	}, pulumi.Provider(args.Provider), pulumi.Parent(parent))
	if err != nil {
		return nil, err
	}
	return cole, nil
}
